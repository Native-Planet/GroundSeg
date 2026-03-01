package broadcast

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/backupsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/leak"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
	"maps"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	broadcastInterval = 1 * time.Second // how often we refresh system info
	broadcastState    structs.AuthBroadcast
	scheduledPacks    = make(map[string]time.Time)
	urbitTransitions  = make(map[string]structs.UrbitTransitionBroadcast)
	schedulePackBus   = make(chan string)
	systemTransitions structs.SystemTransitionBroadcast
	packMu            sync.RWMutex
	urbTransMu        sync.RWMutex
	sysTransMu        sync.RWMutex
	mu                sync.RWMutex // synchronize access to broadcastState
	leakSink          = func(bState structs.AuthBroadcast) {
		leak.LeakChan <- bState
	}
	BackupDir = backupsvc.ResolveBackupRoot(config.BasePath)
)

// SetLeakSinkForTests replaces the side effect used by BroadcastToClients.
// Pass nil to restore the default leak-based sink.
func SetLeakSinkForTests(sink func(structs.AuthBroadcast)) {
	if sink == nil {
		leakSink = func(bState structs.AuthBroadcast) {
			leak.LeakChan <- bState
		}
		return
	}
	leakSink = sink
}

func PublishSchedulePack(reason string) {
	schedulePackBus <- reason
}

func SchedulePackEvents() <-chan string {
	return schedulePackBus
}

func Initialize() error {
	if err := bootstrapBroadcastState(); err != nil {
		return fmt.Errorf("unable to initialize broadcast state: %v", err)
	}
	if err := LoadStartramRegions(); err != nil {
		zap.L().Error("Couldn't load StarTram regions")
	}
	return nil
}

// serialized single thread for ws writes (mutex instead so this isnt necessary)
// func WsDigester() {
// 	for {
// 		event := <-structs.WsEventBus
// 		if event.Conn.Conn != nil {
// 			if err := event.Conn.Conn.WriteMessage(websocket.TextMessage, event.Data); err != nil {
// 				zap.L().Warn(fmt.Sprintf("WS error: %v", err))
// 				if err = auth.WsNilSession(event.Conn.Conn); err != nil {
// 					zap.L().Warn("Couldn't remove WS session")
// 				}
// 				continue
// 			}
// 		}
// 	}
// }

func UpdateScheduledPack(patp string, meldNext time.Time) error {
	packMu.Lock()
	defer packMu.Unlock()
	scheduledPacks[patp] = meldNext
	return nil
}

func GetScheduledPack(patp string) time.Time {
	packMu.Lock()
	defer packMu.Unlock()
	nextPack, exists := scheduledPacks[patp]
	if !exists {
		return time.Time{}
	}
	return nextPack
}

// take in config file and addt'l info to initialize broadcast
func bootstrapBroadcastState() error {
	zap.L().Info("Bootstrapping state")
	// this returns a map of ship:running status
	zap.L().Info("Resolving pier status")
	urbits, err := ConstructPierInfo()
	if err != nil {
		return err
	}
	// update broadcastState with ship info
	mu.Lock()
	broadcastState.Urbits = urbits
	mu.Unlock()
	// update with system state
	sysInfo := constructSystemInfo()
	mu.Lock()
	broadcastState.System = sysInfo
	mu.Unlock()
	// update with profile state
	profileInfo := constructProfileInfo()
	mu.Lock()
	broadcastState.Profile = profileInfo
	mu.Unlock()
	// update with apps state
	appsInfo := constructAppsInfo()
	mu.Lock()
	broadcastState.Apps = appsInfo
	mu.Unlock()
	// start looping info refreshes
	go BroadcastLoop()
	return nil
}

func GetStartramServices() error {
	zap.L().Info("Retrieving StarTram services info")
	if res, err := startram.Retrieve(); err != nil {
		zap.L().Error(fmt.Sprintf("%v", err))
		return err
	} else {
		zap.L().Info(fmt.Sprintf("%+v", res.Subdomains))
		return nil
	}
}

// put startram regions into broadcast struct
func LoadStartramRegions() error {
	zap.L().Info("Retrieving StarTram region info")
	regions, err := startram.SyncRegions()
	if err != nil {
		return err
	} else {
		mu.Lock()
		broadcastState.Profile.Startram.Info.Regions = regions
		mu.Unlock()
	}
	return nil
}

// this is for building the broadcast objects describing piers
func ConstructPierInfo() (map[string]structs.Urbit, error) {
	settings := config.StartramSettingsSnapshot()
	piers := settings.Piers
	updates := make(map[string]structs.Urbit)

	backups := backupSnapshotForPiers(piers, config.GetStartramConfig().Backups)
	runtimeSnapshot, err := runtimeSnapshotForPiers(piers)
	if err != nil {
		errmsg := fmt.Sprintf("Unable to bootstrap urbit states: %v", err)
		zap.L().Error(errmsg)
		return updates, err
	}
	startramSnapshot := startramSnapshotForPiers(config.GetStartramConfig().Subdomains)

	for pier, status := range runtimeSnapshot.pierStatus {
		urbit, ok := assembleUrbitView(
			pier,
			status,
			settings,
			runtimeSnapshot,
			backups,
			startramSnapshot,
		)
		if ok {
			updates[pier] = urbit
		}
	}
	return updates, nil
}

type pierBackupSnapshot struct {
	remote       structs.Backup
	localDaily   structs.Backup
	localWeekly  structs.Backup
	localMonthly structs.Backup
}

type pierRuntimeSnapshot struct {
	currentState structs.AuthBroadcast
	shipNetworks map[string]string
	pierStatus   map[string]string
	hostName     string
}

type pierStartramSnapshot struct {
	remoteReadyByURL map[string]bool
}

func backupSnapshotForPiers(piers []string, remoteBackups []structs.Backup) pierBackupSnapshot {
	return pierBackupSnapshot{
		remote:       flattenRemoteBackups(remoteBackups),
		localDaily:   localBackupsForPeriod(piers, "daily"),
		localWeekly:  localBackupsForPeriod(piers, "weekly"),
		localMonthly: localBackupsForPeriod(piers, "monthly"),
	}
}

func flattenRemoteBackups(remoteBackups []structs.Backup) structs.Backup {
	remoteBackupMap := make(structs.Backup)
	for _, backup := range remoteBackups {
		for ship, backupInfo := range backup {
			remoteBackupMap[ship] = backupInfo
		}
	}
	return remoteBackupMap
}

func localBackupsForPeriod(piers []string, period string) structs.Backup {
	localBackups := make(structs.Backup)
	for _, ship := range piers {
		shipBackups, err := filepath.Glob(filepath.Join(BackupDir, ship, period, "*"))
		if err != nil {
			continue
		}
		for _, backup := range shipBackups {
			timestamp, err := strconv.Atoi(filepath.Base(backup))
			if err != nil {
				continue
			}
			localBackups[ship] = append(localBackups[ship], structs.BackupObject{Timestamp: timestamp, MD5: ""})
		}
	}
	return localBackups
}

func runtimeSnapshotForPiers(piers []string) (pierRuntimeSnapshot, error) {
	return runtimeSnapshotCollector{}.collect(piers)
}

func resolveBroadcastHostName() string {
	hostName := system.LocalUrl
	if hostName == "" {
		zap.L().Debug("Defaulting to `nativeplanet.local`")
		hostName = "nativeplanet.local"
	}
	return hostName
}

func startramSnapshotForPiers(subdomains []structs.Subdomain) pierStartramSnapshot {
	readyByURL := make(map[string]bool, len(subdomains))
	for _, subdomain := range subdomains {
		readyByURL[subdomain.URL] = subdomain.Status == "ok"
	}
	return pierStartramSnapshot{
		remoteReadyByURL: readyByURL,
	}
}

func assembleUrbitView(
	pier string,
	status string,
	settings config.StartramSettings,
	runtimeSnapshot pierRuntimeSnapshot,
	backups pierBackupSnapshot,
	startramSnapshot pierStartramSnapshot,
) (structs.Urbit, bool) {
	return urbitViewMapper{}.assemble(
		pier,
		status,
		settings,
		runtimeSnapshot,
		backups,
		startramSnapshot,
	)
}

func lusCodeIfRunning(pier string, status string) string {
	if strings.Contains(status, "Up") {
		lusCode, _ := click.GetLusCode(pier)
		return lusCode
	}
	return ""
}

func deskInstalledIfRunning(pier string, status string, desk string) bool {
	if !strings.Contains(status, "Up") {
		return false
	}
	deskStatus, err := click.GetDesk(pier, desk, false)
	if err != nil {
		zap.L().Debug(fmt.Sprintf("Broadcast failed to get %s desk info for %v: %v", desk, pier, err))
		return false
	}
	return deskStatus == "running"
}

func boolSettingWithDefaultTrue(setting any) bool {
	if value, ok := setting.(bool); ok {
		return value
	}
	return true
}

func normalizePackSchedule(meldDay string, meldDate int) (string, int) {
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	packDay := "Monday"
	for _, day := range days {
		if day == meldDay {
			packDay = strings.Title(meldDay)
			break
		}
	}
	packDate := 1
	if meldDate > 1 {
		packDate = meldDate
	}
	return packDay, packDate
}

func constructAppsInfo() structs.Apps {
	return appInfoCollector{}.collect()
}

func constructProfileInfo() structs.Profile {
	return profileInfoCollector{}.collect()
}

// put together the system[usage] subobject
func constructSystemInfo() structs.System {
	return systemInfoCollector{}.collect()
}

// return a map of ships and their networks
func GetContainerNetworks(containers []string) map[string]string {
	res := make(map[string]string)
	for _, container := range containers {
		network, err := docker.GetContainerNetwork(container)
		if err != nil {
			//errmsg := fmt.Sprintf("Error getting container network: %v", err)
			//zap.L().Error(errmsg) // temp surpress
			continue
		} else {
			res[container] = network
		}
	}
	return res
}

// stupid update method instead of psychotic recursion
func UpdateBroadcast(broadcast structs.AuthBroadcast) {
	mu.Lock()
	defer mu.Unlock()
	broadcastState = broadcast
}

// return broadcast state
func GetState() structs.AuthBroadcast {
	mu.RLock()
	defer mu.RUnlock()
	return cloneBroadcastState(broadcastState)
}

func cloneBroadcastState(in structs.AuthBroadcast) structs.AuthBroadcast {
	out := in
	if in.Urbits != nil {
		out.Urbits = make(map[string]structs.Urbit, len(in.Urbits))
		for patp, urbit := range in.Urbits {
			cloned := urbit
			cloned.Info.RemoteTlonBackups = append([]structs.BackupObject(nil), urbit.Info.RemoteTlonBackups...)
			cloned.Info.LocalDailyTlonBackups = append([]structs.BackupObject(nil), urbit.Info.LocalDailyTlonBackups...)
			cloned.Info.LocalWeeklyTlonBackups = append([]structs.BackupObject(nil), urbit.Info.LocalWeeklyTlonBackups...)
			cloned.Info.LocalMonthlyTlonBackups = append([]structs.BackupObject(nil), urbit.Info.LocalMonthlyTlonBackups...)
			out.Urbits[patp] = cloned
		}
	}
	if in.System.Info.Drives != nil {
		out.System.Info.Drives = maps.Clone(in.System.Info.Drives)
	}
	if in.System.Info.SMART != nil {
		out.System.Info.SMART = maps.Clone(in.System.Info.SMART)
	}
	if in.System.Info.Usage.Disk != nil {
		out.System.Info.Usage.Disk = maps.Clone(in.System.Info.Usage.Disk)
	}
	out.System.Info.Usage.RAM = append([]uint64(nil), in.System.Info.Usage.RAM...)
	out.System.Info.Wifi.Networks = append([]string(nil), in.System.Info.Wifi.Networks...)
	if in.Profile.Startram.Info.Regions != nil {
		out.Profile.Startram.Info.Regions = maps.Clone(in.Profile.Startram.Info.Regions)
	}
	out.Profile.Startram.Info.StartramServices = append([]string(nil), in.Profile.Startram.Info.StartramServices...)
	out.System.Transition.Error = append([]string(nil), in.System.Transition.Error...)
	out.Logs.Containers.Wireguard.Logs = append([]any(nil), in.Logs.Containers.Wireguard.Logs...)
	out.Logs.System.Logs = append([]any(nil), in.Logs.System.Logs...)
	out.Apps.Penpai.Info.Models = append([]string(nil), in.Apps.Penpai.Info.Models...)
	return out
}

// return json string of current broadcast state
func GetStateJson(bState structs.AuthBroadcast) ([]byte, error) {
	//temp
	bState.Type = "structure"
	bState.AuthLevel = "authorized"
	//end temp
	broadcastJson, err := json.Marshal(bState)
	if err != nil {
		errmsg := fmt.Sprintf("Error marshalling response: %v", err)
		zap.L().Error(errmsg)
		return nil, err
	}
	return broadcastJson, nil
}

// broadcast the global state to auth'd clients
func BroadcastToClients() error {
	bState := GetState()
	leakSink(bState) // vere 3.0
	cm := auth.GetClientManager()
	if cm.HasAuthSession() {
		authJson, err := GetStateJson(bState)
		auth.ClientManager.BroadcastAuth(authJson)
		if err != nil {
			return err
		}
		auth.ClientManager.BroadcastAuth(authJson)
		return nil
	}
	return nil
}

// broadcast to unauth clients
func UnauthBroadcast(input []byte) error {
	auth.ClientManager.BroadcastUnauth(input)
	return nil
}
func ReloadUrbits() error {
	zap.L().Info("Reloading ships in broadcast")
	urbits, err := ConstructPierInfo()
	if err != nil {
		return err
	}
	mu.Lock()
	broadcastState.Urbits = urbits
	mu.Unlock()
	return nil
}
