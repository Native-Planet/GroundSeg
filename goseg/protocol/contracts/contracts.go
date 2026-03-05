package contracts

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ContractCompatibility models a contract's compatibility policy across versions.
type ContractCompatibility string

const (
	CompatibilityStable       ContractCompatibility = "stable"
	CompatibilityDeprecated   ContractCompatibility = "deprecated"
	CompatibilityRemoved      ContractCompatibility = "removed"
	CompatibilityBackwardSafe ContractCompatibility = "backward-compatible"
)

// ContractMetadata captures lifecycle and compatibility information for a contract.
type ContractMetadata struct {
	IntroducedIn  string
	DeprecatedIn  string
	RemovedIn     string
	Compatibility ContractCompatibility
}

// ContractID identifies a contract in the shared governance catalog.
type ContractID string

// ActionVerb identifies the wire-level action token string for each action binding.
type ActionVerb string

// ActionNamespace identifies the protocol namespace for action contracts.
type ActionNamespace string

// ActionContractBinding joins an action identifier with its contract descriptor ID
// and namespace, allowing runtime validation and single-source discovery.
type ActionContractBinding struct {
	Namespace ActionNamespace
	Action    ActionVerb
	Contract  ContractID
}

// ContractDescriptor captures both lifecycle metadata and human-facing contract details.
type ContractDescriptor struct {
	Name        string
	Description string
	ContractMetadata
	Message string
}

func (contract ContractDescriptor) IsDeprecated(version string) bool {
	return IsContractDeprecated(version, contract.ContractMetadata)
}

func (contract ContractDescriptor) IsActive(version string) bool {
	return IsContractActive(version, contract.ContractMetadata)
}

type contractCatalogEntry struct {
	ID         ContractID
	Namespace  ActionNamespace
	Action     ActionVerb
	Governance contractGovernanceMetadata
	Descriptor ContractDescriptor
}

type ContractCatalogEntry = contractCatalogEntry

type actionContractBindingKey struct {
	Namespace ActionNamespace
	Action    ActionVerb
}

type contractCatalogFamily struct {
	name     string
	specs    func() []contractCatalogEntry
	validate func([]contractCatalogEntry) error
}

func protocolActionContractID(namespace ActionNamespace, action ActionVerb) ContractID {
	base := contractID("protocol", "actions", string(namespace), string(action))
	return ContractID(base)
}

func contractID(parts ...string) ContractID {
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			segments = append(segments, trimmed)
		}
	}
	return ContractID(strings.Join(segments, "."))
}

// ContractCatalog owns all contract descriptors and action-to-contract bindings.
type ContractCatalog struct {
	contractCatalog             map[ContractID]ContractDescriptor
	actionContractBindings      []ActionContractBinding
	actionContractBindingsByKey map[actionContractBindingKey]ContractID
}

var contractCatalogFamilySpecs = []contractCatalogFamily{
	{name: "protocol", specs: protocolContractCatalogSpecs, validate: validateProtocolContractSpecs},
	{name: "startram", specs: startramContractCatalogSpecs, validate: validateStartramContractSpecs},
}

const (
	contractCatalogFamilyProtocol = "protocol"
	contractCatalogFamilyStartram = "startram"
)

func contractCatalogEntriesForFamily(name string) ([]contractCatalogEntry, bool) {
	for _, family := range contractCatalogFamilySpecs {
		if family.name != name {
			continue
		}
		return catalogEntries(family.specs()), true
	}
	return nil, false
}

func contractCatalogEntriesForFamilySnapshot(name string) ([]contractCatalogEntry, bool) {
	for _, family := range contractCatalogFamilySpecs {
		if family.name != name {
			continue
		}
		return catalogEntriesSnapshot(family.specs()), true
	}
	return nil, false
}

func catalogEntries(specs []contractCatalogEntry) []contractCatalogEntry {
	out := make([]contractCatalogEntry, len(specs))
	copy(out, specs)
	return out
}

func catalogEntriesSnapshot(specs []contractCatalogEntry) []contractCatalogEntry {
	return catalogEntries(specs)
}

// ContractCatalogEntries returns the canonical catalog source list used by LoadRegistry.
func ContractCatalogEntries() []ContractCatalogEntry {
	entries := make([]ContractCatalogEntry, 0, 16)
	for _, family := range contractCatalogFamilySpecs {
		familyEntries := catalogEntries(family.specs())
		if family.validate != nil {
			if err := family.validate(familyEntries); err != nil {
				panic(fmt.Sprintf("invalid %s contract family specs: %v", family.name, err))
			}
		}
		entries = append(entries, familyEntries...)
	}
	return entries
}

// ContractCatalogEntriesForFamily returns a snapshot of canonical contract catalog
// entries for a named family.
func ContractCatalogEntriesForFamily(name string) ([]ContractCatalogEntry, bool) {
	entries, ok := contractCatalogEntriesForFamilySnapshot(name)
	if !ok {
		return nil, false
	}
	out := make([]ContractCatalogEntry, len(entries))
	copy(out, entries)
	return out, true
}

// ProtocolContractCatalogEntries returns canonical protocol action contract entries.
func ProtocolContractCatalogEntries() []ContractCatalogEntry {
	entries, ok := ContractCatalogEntriesForFamily(contractCatalogFamilyProtocol)
	if !ok {
		return []ContractCatalogEntry{}
	}
	return entries
}

// StartramContractCatalogEntries returns canonical Startram contract entries.
func StartramContractCatalogEntries() []ContractCatalogEntry {
	entries, ok := ContractCatalogEntriesForFamily(contractCatalogFamilyStartram)
	if !ok {
		return []ContractCatalogEntry{}
	}
	return entries
}

func validateFamilyCatalogEntries(family string, entries []contractCatalogEntry) error {
	if _, err := NewCatalogFromEntries(entries); err != nil {
		return fmt.Errorf("validate %s contract catalog: %w", family, err)
	}
	return nil
}

// NewCatalogFromEntries validates and builds a contract catalog from supplied entries.
func NewCatalogFromEntries(entries []contractCatalogEntry) (*ContractCatalog, error) {
	nextCatalog := make(map[ContractID]ContractDescriptor, len(entries))
	nextBindings := make([]ActionContractBinding, 0, len(entries))
	nextBindingByKey := make(map[actionContractBindingKey]ContractID, len(entries))
	seenContracts := make(map[ContractID]struct{})
	seenBindings := make(map[actionContractBindingKey]struct{})

	for _, entry := range entries {
		if err := validateContractCatalogIdentity(entry); err != nil {
			return nil, fmt.Errorf("%w: contract %s catalog integrity: %w", errMissingContractDescriptor, entry.ID, err)
		}
		if entry.ID == "" {
			return nil, fmt.Errorf("%w: missing contract id", errMissingContractDescriptor)
		}
		if _, ok := seenContracts[entry.ID]; ok {
			return nil, fmt.Errorf("%w: duplicate contract id %q", errMissingContractDescriptor, entry.ID)
		}
		seenContracts[entry.ID] = struct{}{}

		if err := validateContractMetadata(entry.ID, entry.Descriptor.ContractMetadata); err != nil {
			return nil, fmt.Errorf("%w: contract %s metadata: %w", errMissingContractDescriptor, entry.ID, err)
		}
		if entry.Descriptor.Name == "" {
			return nil, fmt.Errorf("%w: contract %s missing name", errMissingContractDescriptor, entry.ID)
		}
		if entry.Descriptor.Description == "" {
			return nil, fmt.Errorf("%w: contract %s missing description", errMissingContractDescriptor, entry.ID)
		}

		nextCatalog[entry.ID] = entry.Descriptor

		if (entry.Namespace == "") != (entry.Action == "") {
			return nil, fmt.Errorf("%w: contract %s binding missing namespace or action", errMissingContractDescriptor, entry.ID)
		}
		if entry.Namespace == "" {
			continue
		}

		binding := ActionContractBinding{
			Namespace: entry.Namespace,
			Action:    entry.Action,
			Contract:  entry.ID,
		}
		bindingKey := actionContractBindingKey{
			Namespace: binding.Namespace,
			Action:    binding.Action,
		}
		if _, ok := seenBindings[bindingKey]; ok {
			return nil, fmt.Errorf("%w: duplicate action binding %s:%s", errMissingContractDescriptor, binding.Namespace, binding.Action)
		}
		nextBindings = append(nextBindings, binding)
		nextBindingByKey[bindingKey] = binding.Contract
		seenBindings[bindingKey] = struct{}{}
	}

	sort.Slice(nextBindings, func(i, j int) bool {
		if nextBindings[i].Namespace == nextBindings[j].Namespace {
			return nextBindings[i].Action < nextBindings[j].Action
		}
		return nextBindings[i].Namespace < nextBindings[j].Namespace
	})

	return &ContractCatalog{
		contractCatalog:             nextCatalog,
		actionContractBindings:      nextBindings,
		actionContractBindingsByKey: nextBindingByKey,
	}, nil
}

func validateContractCatalogIdentity(entry contractCatalogEntry) error {
	if entry.Namespace == "" && entry.Action != "" {
		return fmt.Errorf("binding has action %q without namespace", entry.Action)
	}
	if entry.Namespace != "" {
		if entry.Action == "" {
			return fmt.Errorf("binding has namespace %q without action", entry.Namespace)
		}
		if !strings.HasPrefix(string(entry.ID), "protocol.actions.") {
			return fmt.Errorf("protocol action namespace %q requires protocol.actions prefix", entry.Namespace)
		}
		namespacePrefix := fmt.Sprintf("protocol.actions.%s.", entry.Namespace)
		if !strings.HasPrefix(string(entry.ID), namespacePrefix) {
			return fmt.Errorf("protocol action binding namespace mismatch: expected %q prefix in %q", namespacePrefix, entry.ID)
		}
		if !strings.HasSuffix(string(entry.ID), "."+string(entry.Action)) {
			return fmt.Errorf("protocol action %s should be reflected in contract id %q", entry.Action, entry.ID)
		}
		if expected := protocolActionContractID(entry.Namespace, entry.Action); expected != entry.ID {
			return fmt.Errorf("protocol action %s in namespace %s drifted contract id: expected %s", entry.Action, entry.Namespace, expected)
		}
	}
	if entry.ID == "" {
		return fmt.Errorf("missing contract id")
	}
	return nil
}

func (catalog *ContractCatalog) copy() *ContractCatalog {
	if catalog == nil {
		return nil
	}
	nextCatalog := make(map[ContractID]ContractDescriptor, len(catalog.contractCatalog))
	for id, descriptor := range catalog.contractCatalog {
		nextCatalog[id] = descriptor
	}
	nextBindings := make([]ActionContractBinding, len(catalog.actionContractBindings))
	copy(nextBindings, catalog.actionContractBindings)
	nextBindingByKey := make(map[actionContractBindingKey]ContractID, len(catalog.actionContractBindingsByKey))
	for key, value := range catalog.actionContractBindingsByKey {
		nextBindingByKey[key] = value
	}
	return &ContractCatalog{
		contractCatalog:             nextCatalog,
		actionContractBindings:      nextBindings,
		actionContractBindingsByKey: nextBindingByKey,
	}
}

// ContractDescriptorFor looks up a contract descriptor by ID in this catalog.
func (catalog *ContractCatalog) ContractDescriptorFor(id ContractID) (ContractDescriptor, bool) {
	if catalog == nil {
		return ContractDescriptor{}, false
	}
	descriptor, ok := catalog.contractCatalog[id]
	return descriptor, ok
}

// ContractDescriptorsForDebug returns a copy of the contract map for deterministic test
// inspection without exposing internal mutation.
func (catalog *ContractCatalog) ContractDescriptorsForDebug() map[ContractID]ContractDescriptor {
	if catalog == nil {
		return map[ContractID]ContractDescriptor{}
	}
	return catalog.contractCatalog
}

// ActionContractFor returns the action contract descriptor for a namespaced action.
func (catalog *ContractCatalog) ActionContractFor(namespace ActionNamespace, action ActionVerb) (ContractDescriptor, bool) {
	binding, ok := catalog.ActionContractBindingFor(namespace, action)
	if !ok {
		return ContractDescriptor{}, false
	}
	return catalog.ContractDescriptorFor(binding.Contract)
}

// ActionContractBindingFor returns the binding for a namespaced action.
func (catalog *ContractCatalog) ActionContractBindingFor(namespace ActionNamespace, action ActionVerb) (ActionContractBinding, bool) {
	if catalog == nil {
		return ActionContractBinding{}, false
	}
	key := actionContractBindingKey{
		Namespace: namespace,
		Action:    action,
	}
	contractID, ok := catalog.actionContractBindingsByKey[key]
	if !ok {
		return ActionContractBinding{}, false
	}
	return ActionContractBinding{Namespace: namespace, Action: action, Contract: contractID}, true
}

// ActionContractBindings exposes canonical action-to-contract bindings as an immutable copy.
func (catalog *ContractCatalog) ActionContractBindings() []ActionContractBinding {
	if catalog == nil {
		return nil
	}
	out := make([]ActionContractBinding, len(catalog.actionContractBindings))
	copy(out, catalog.actionContractBindings)
	return out
}

// ActionContractBindingsForNamespace returns canonical action-to-contract bindings in a namespace.
func (catalog *ContractCatalog) ActionContractBindingsForNamespace(namespace ActionNamespace) []ActionContractBinding {
	if catalog == nil {
		return nil
	}
	out := make([]ActionContractBinding, 0, len(catalog.actionContractBindings))
	for _, binding := range catalog.actionContractBindings {
		if binding.Namespace == namespace {
			out = append(out, binding)
		}
	}
	return out
}

// ValidateActionContractBindings validates bindings against the registry.
func (catalog *ContractCatalog) ValidateActionContractBindings() error {
	if catalog == nil {
		return fmt.Errorf("%w: registry not initialized", errMissingContractDescriptor)
	}
	seenBindingKeys := map[actionContractBindingKey]struct{}{}
	for _, binding := range catalog.actionContractBindings {
		if binding.Namespace == "" {
			return fmt.Errorf("%w: missing namespace for action %s", errMissingContractDescriptor, binding.Action)
		}
		if binding.Action == "" {
			return fmt.Errorf("%w: missing action in namespace %s", errMissingContractDescriptor, binding.Namespace)
		}
		key := actionContractBindingKey{Namespace: binding.Namespace, Action: binding.Action}
		if _, seenBefore := seenBindingKeys[key]; seenBefore {
			return fmt.Errorf("%w: duplicate action binding %s:%s", errMissingContractDescriptor, binding.Namespace, binding.Action)
		}
		seenBindingKeys[key] = struct{}{}

		descriptor, ok := catalog.ContractDescriptorFor(binding.Contract)
		if !ok {
			return fmt.Errorf("%w: missing contract %s for action %s:%s", errMissingContractDescriptor, binding.Contract, binding.Namespace, binding.Action)
		}
		if descriptor.Name == "" {
			return fmt.Errorf("%w: contract %s has missing name", errMissingContractDescriptor, binding.Contract)
		}
		if descriptor.Description == "" {
			return fmt.Errorf("%w: contract %s has missing description", errMissingContractDescriptor, binding.Contract)
		}
	}
	return nil
}

var (
	errMissingContractDescriptor = errors.New("contract descriptor lookup failed")

	defaultRegistry     *ContractCatalog
	defaultRegistryErr  error
	defaultRegistryOnce sync.Once
)

// LoadRegistry composes protocol and startram contract entries into a validated catalog.
func LoadRegistry() (*ContractCatalog, error) {
	return NewCatalogFromEntries(ContractCatalogEntries())
}

func defaultRegistrySnapshot() (*ContractCatalog, error) {
	defaultRegistryOnce.Do(func() {
		defaultRegistry, defaultRegistryErr = LoadRegistry()
	})
	if defaultRegistryErr != nil {
		return nil, defaultRegistryErr
	}
	return defaultRegistry.copy(), nil
}

// ContractDescriptorFor looks up a contract descriptor by ID from the default catalog.
//
// This boolean API is intentionally lossy: it returns false for both missing IDs and
// transient catalog initialization failures.
// Use ContractDescriptorForWithError for explicit diagnostics when needed.
func ContractDescriptorFor(name ContractID) (ContractDescriptor, bool) {
	registry, err := defaultRegistrySnapshot()
	if err != nil {
		return ContractDescriptor{}, false
	}
	contract, ok := registry.ContractDescriptorFor(name)
	return contract, ok
}

// ContractCatalogInitError returns the current default catalog initialization error, if any.
// If initialization succeeds this returns nil.
func ContractCatalogInitError() error {
	_, err := defaultRegistrySnapshot()
	return err
}

// ContractDescriptorForWithError looks up a contract descriptor and surfaces catalog errors.
func ContractDescriptorForWithError(name ContractID) (ContractDescriptor, bool, error) {
	registry, err := defaultRegistrySnapshot()
	if err != nil {
		return ContractDescriptor{}, false, err
	}
	contract, ok := registry.ContractDescriptorFor(name)
	return contract, ok, nil
}

// MustContractDescriptor panics if the contract ID is missing or the registry cannot initialize.
func MustContractDescriptor(name ContractID) ContractDescriptor {
	contract, ok, err := ContractDescriptorForWithError(name)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize protocol contracts: %v", err))
	}
	if !ok {
		panic(fmt.Sprintf("missing contract descriptor for %s", name))
	}
	return contract
}

// ActionContractFor returns the action contract descriptor for a namespaced action.
func ActionContractFor(namespace ActionNamespace, action ActionVerb) (ContractDescriptor, bool) {
	registry, err := defaultRegistrySnapshot()
	if err != nil {
		return ContractDescriptor{}, false
	}
	return registry.ActionContractFor(namespace, action)
}

// ActionContractForBinding returns the action contract descriptor for a binding.
func ActionContractForBinding(binding ActionContractBinding) (ContractDescriptor, bool) {
	return ActionContractFor(binding.Namespace, binding.Action)
}

// ActionContractBindings exposes canonical action-to-contract bindings as an immutable copy.
func ActionContractBindings() []ActionContractBinding {
	registry, err := defaultRegistrySnapshot()
	if err != nil {
		return nil
	}
	return registry.ActionContractBindings()
}

// ActionContractBindingsForNamespace returns action bindings in a namespace.
func ActionContractBindingsForNamespace(namespace ActionNamespace) []ActionContractBinding {
	registry, err := defaultRegistrySnapshot()
	if err != nil {
		return nil
	}
	return registry.ActionContractBindingsForNamespace(namespace)
}

// ValidateActionContractBindings validates the loaded action-to-contract bindings.
func ValidateActionContractBindings() error {
	registry, err := defaultRegistrySnapshot()
	if err != nil {
		return err
	}
	return registry.ValidateActionContractBindings()
}

// ValidateGovernanceSchema validates family-level governance specs and combined catalog integrity.
func ValidateGovernanceSchema() error {
	entries := make([]contractCatalogEntry, 0, 16)
	for _, family := range contractCatalogFamilySpecs {
		familyEntries := catalogEntries(family.specs())
		if family.validate != nil {
			if err := family.validate(familyEntries); err != nil {
				return fmt.Errorf("validate %s governance schema: %w", family.name, err)
			}
		}
		entries = append(entries, familyEntries...)
	}
	if _, err := NewCatalogFromEntries(entries); err != nil {
		return fmt.Errorf("validate combined governance schema: %w", err)
	}
	return nil
}

func validateContractMetadata(contractID ContractID, metadata ContractMetadata) error {
	if metadata.IntroducedIn == "" {
		return fmt.Errorf("%s has no introduced version", contractID)
	}
	introduced, err := time.Parse(contractVersionLayout, metadata.IntroducedIn)
	if err != nil {
		return fmt.Errorf("%s has invalid introduced version %q: %w", contractID, metadata.IntroducedIn, err)
	}

	if !isKnownCompatibility(metadata.Compatibility) {
		return fmt.Errorf("%s has unknown compatibility %q", contractID, metadata.Compatibility)
	}

	switch metadata.Compatibility {
	case CompatibilityDeprecated:
		if metadata.DeprecatedIn == "" {
			return fmt.Errorf("%s has deprecated compatibility without deprecated version", contractID)
		}
	case CompatibilityRemoved:
		if metadata.RemovedIn == "" {
			return fmt.Errorf("%s has removed compatibility without removed version", contractID)
		}
	case CompatibilityStable, CompatibilityBackwardSafe:
		if metadata.DeprecatedIn != "" || metadata.RemovedIn != "" {
			return fmt.Errorf("%s has compatibility %q with lifecycle versions set", contractID, metadata.Compatibility)
		}
	}

	if metadata.DeprecatedIn != "" {
		deprecated, err := time.Parse(contractVersionLayout, metadata.DeprecatedIn)
		if err != nil {
			return fmt.Errorf("%s has invalid deprecated version %q: %w", contractID, metadata.DeprecatedIn, err)
		}
		if deprecated.Before(introduced) {
			return fmt.Errorf("%s has deprecated before introduced date", contractID)
		}
	}

	if metadata.RemovedIn != "" {
		removed, err := time.Parse(contractVersionLayout, metadata.RemovedIn)
		if err != nil {
			return fmt.Errorf("%s has invalid removed version %q: %w", contractID, metadata.RemovedIn, err)
		}
		if removed.Before(introduced) {
			return fmt.Errorf("%s has removed before introduced date", contractID)
		}
		if metadata.DeprecatedIn != "" && removed.Before(timeMustParse("2006.01.02", metadata.DeprecatedIn)) {
			return fmt.Errorf("%s has removed-before-deprecated date", contractID)
		}
	}
	return nil
}

func isKnownCompatibility(value ContractCompatibility) bool {
	switch value {
	case CompatibilityStable, CompatibilityDeprecated, CompatibilityRemoved, CompatibilityBackwardSafe:
		return true
	default:
		return false
	}
}

func timeMustParse(layout, value string) time.Time {
	parsed, _ := time.Parse(layout, value)
	return parsed
}

const contractVersionLayout = "2006.01.02"

// CurrentContractVersion is the canonical active contract version for compatibility checks.
const CurrentContractVersion = "2026.03.02"

// IsVersionAtLeast reports whether version is at or after target.
func IsVersionAtLeast(version string, target string) bool {
	if target == "" {
		return false
	}
	v, err := time.Parse(contractVersionLayout, version)
	if err != nil {
		return false
	}
	cutoff, err := time.Parse(contractVersionLayout, target)
	if err != nil {
		return false
	}
	return !v.Before(cutoff)
}

// IsVersionAtLeastOrEqual is an alias for IsVersionAtLeast for readability.
func IsVersionAtLeastOrEqual(version string, target string) bool {
	return IsVersionAtLeast(version, target)
}

// IsContractDeprecated reports whether the version is within the explicit deprecated
// window and not yet removed.
func IsContractDeprecated(version string, metadata ContractMetadata) bool {
	return metadata.DeprecatedIn != "" &&
		IsVersionAtLeast(version, metadata.DeprecatedIn) &&
		!IsVersionAtLeastOrEqual(version, metadata.RemovedIn)
}

// IsContractActive reports whether the contract is introduced and not yet removed at version.
func IsContractActive(version string, metadata ContractMetadata) bool {
	if !IsVersionAtLeast(version, metadata.IntroducedIn) {
		return false
	}
	return !IsVersionAtLeastOrEqual(version, metadata.RemovedIn)
}
