package defaults

import (
	"encoding/json"
	"fmt"
	"goseg/structs"
	"log/slog"
	"os"
)

var (
	logger             = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	DefaultVersionText = `{
		"groundseg": {
		  "canary": {
			"groundseg": {
			  "amd64_sha256": "58041cd66c692cddd09e83dedf6c803c1a1a5a46a6cc28167a3fb3e736f3e160",
			  "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v1.4.2_latest",
			  "arm64_sha256": "4256fc53658a3bfcde793157951ea587c0992a0de879d8eb016a941097163e47",
			  "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v1.4.2_latest",
			  "major": 1,
			  "minor": 4,
			  "patch": 2
			},
			"manual": {
			  "amd64_sha256": "148a2acb946c4c38720cf8994a39b20f655547fd43996f2449ff5b7bc24793c9",
			  "arm64_sha256": "baa87ad152ec14a7df9cd889e970cd1ad9c5f6bbc0e1fbd587f5ed6c89e31f08",
			  "repo": "registry.hub.docker.com/nativeplanet/groundseg-manual",
			  "tag": "latest"
			},
			"minio": {
			  "amd64_sha256": "f6a3001a765dc59a8e365149ade0ea628494230e984891877ead016eb24ba9a9",
			  "arm64_sha256": "567779c9f29aca670f84d066051290faeaae6c3ad3a3b7062de4936aaab2a29d",
			  "repo": "registry.hub.docker.com/minio/minio",
			  "tag": "latest"
			},
			"miniomc": {
			  "amd64_sha256": "6ffd76764e8ca484de12c6ecaa352db3d8efd5c9d44f393718b29b6600e0a559",
			  "arm64_sha256": "6825aecd2f123c9d4408e660aba8a72f9e547a3774350b8f4d2d9b674e99e424",
			  "repo": "registry.hub.docker.com/minio/mc",
			  "tag": "latest"
			},
			"netdata": {
			  "amd64_sha256": "95e74c36f15091bcd7983ee162248f1f91c21207c235fce6b0d6f8ed9a11732a",
			  "arm64_sha256": "cd3dc9d182a4561b162f03c6986f4647bbb704f8e7e4872ee0611b1b9e86e1b0",
			  "repo": "registry.hub.docker.com/netdata/netdata",
			  "tag": "latest"
			},
			"vere": {
			  "amd64_sha256": "674a37a736883275991aa094c7b58e1acad4ab4a15b6aeb896d5c72016faa58f",
			  "arm64_sha256": "None",
			  "repo": "registry.hub.docker.com/nativeplanet/urbit",
			  "tag": "edge"
			},
			"webui": {
			  "amd64_sha256": "a2c683530d15b1095bcb3e97b1b6236461142e6e19774762991e30c817352797",
			  "arm64_sha256": "0b96f7a02efc0d5c754bd6d89df2b9f738a676c51b529258b99d5b84e30d1725",
			  "repo": "registry.hub.docker.com/nativeplanet/groundseg-webui",
			  "tag": "latest"
			},
			"wireguard": {
			  "amd64_sha256": "ae6f8e8cc1303bc9c0b5fa1b1ef4176c25a2c082e29bf8b554ce1196731e7db2",
			  "arm64_sha256": "403d741b1b5bcf5df1e48eab0af8038355fae3e29419ad5980428f9aebd1576c",
			  "repo": "registry.hub.docker.com/linuxserver/wireguard",
			  "tag": "latest"
			}
		  },
		  "edge": {
			"groundseg": {
			  "amd64_sha256": "58041cd66c692cddd09e83dedf6c803c1a1a5a46a6cc28167a3fb3e736f3e160",
			  "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v1.4.2_edge",
			  "arm64_sha256": "4256fc53658a3bfcde793157951ea587c0992a0de879d8eb016a941097163e47",
			  "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v1.4.2_edge",
			  "major": 1,
			  "minor": 4,
			  "patch": 2
			},
			"manual": {
			  "amd64_sha256": "148a2acb946c4c38720cf8994a39b20f655547fd43996f2449ff5b7bc24793c9",
			  "arm64_sha256": "baa87ad152ec14a7df9cd889e970cd1ad9c5f6bbc0e1fbd587f5ed6c89e31f08",
			  "repo": "registry.hub.docker.com/nativeplanet/groundseg-manual",
			  "tag": "edge"
			},
			"minio": {
			  "amd64_sha256": "f6a3001a765dc59a8e365149ade0ea628494230e984891877ead016eb24ba9a9",
			  "arm64_sha256": "567779c9f29aca670f84d066051290faeaae6c3ad3a3b7062de4936aaab2a29d",
			  "repo": "registry.hub.docker.com/minio/minio",
			  "tag": "latest"
			},
			"miniomc": {
			  "amd64_sha256": "6ffd76764e8ca484de12c6ecaa352db3d8efd5c9d44f393718b29b6600e0a559",
			  "arm64_sha256": "6825aecd2f123c9d4408e660aba8a72f9e547a3774350b8f4d2d9b674e99e424",
			  "repo": "registry.hub.docker.com/minio/mc",
			  "tag": "latest"
			},
			"netdata": {
			  "amd64_sha256": "95e74c36f15091bcd7983ee162248f1f91c21207c235fce6b0d6f8ed9a11732a",
			  "arm64_sha256": "cd3dc9d182a4561b162f03c6986f4647bbb704f8e7e4872ee0611b1b9e86e1b0",
			  "repo": "registry.hub.docker.com/netdata/netdata",
			  "tag": "latest"
			},
			"vere": {
			  "amd64_sha256": "7dc0a1f97214101482d9c329a5108471bcf23fafea421e1ae2662c6c20377037",
			  "arm64_sha256": "1dbded539bd99cd789bfe5cbe11bb89baa598e213881389e8138da8fc0a27fa9",
			  "repo": "registry.hub.docker.com/nativeplanet/urbit",
			  "tag": "v2.11"
			},
			"webui": {
			  "amd64_sha256": "a2c683530d15b1095bcb3e97b1b6236461142e6e19774762991e30c817352797",
			  "arm64_sha256": "0b96f7a02efc0d5c754bd6d89df2b9f738a676c51b529258b99d5b84e30d1725",
			  "repo": "registry.hub.docker.com/nativeplanet/groundseg-webui",
			  "tag": "edge"
			},
			"wireguard": {
			  "amd64_sha256": "ae6f8e8cc1303bc9c0b5fa1b1ef4176c25a2c082e29bf8b554ce1196731e7db2",
			  "arm64_sha256": "403d741b1b5bcf5df1e48eab0af8038355fae3e29419ad5980428f9aebd1576c",
			  "repo": "registry.hub.docker.com/linuxserver/wireguard",
			  "tag": "latest"
			}
		  },
		  "latest": {
			"groundseg": {
			  "amd64_sha256": "58041cd66c692cddd09e83dedf6c803c1a1a5a46a6cc28167a3fb3e736f3e160",
			  "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v1.4.2_latest",
			  "arm64_sha256": "4256fc53658a3bfcde793157951ea587c0992a0de879d8eb016a941097163e47",
			  "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v1.4.2_latest",
			  "major": 1,
			  "minor": 4,
			  "patch": 2
			},
			"manual": {
			  "amd64_sha256": "148a2acb946c4c38720cf8994a39b20f655547fd43996f2449ff5b7bc24793c9",
			  "arm64_sha256": "baa87ad152ec14a7df9cd889e970cd1ad9c5f6bbc0e1fbd587f5ed6c89e31f08",
			  "repo": "registry.hub.docker.com/nativeplanet/groundseg-manual",
			  "tag": "latest"
			},
			"minio": {
			  "amd64_sha256": "f6a3001a765dc59a8e365149ade0ea628494230e984891877ead016eb24ba9a9",
			  "arm64_sha256": "567779c9f29aca670f84d066051290faeaae6c3ad3a3b7062de4936aaab2a29d",
			  "repo": "registry.hub.docker.com/minio/minio",
			  "tag": "latest"
			},
			"miniomc": {
			  "amd64_sha256": "6ffd76764e8ca484de12c6ecaa352db3d8efd5c9d44f393718b29b6600e0a559",
			  "arm64_sha256": "6825aecd2f123c9d4408e660aba8a72f9e547a3774350b8f4d2d9b674e99e424",
			  "repo": "registry.hub.docker.com/minio/mc",
			  "tag": "latest"
			},
			"netdata": {
			  "amd64_sha256": "95e74c36f15091bcd7983ee162248f1f91c21207c235fce6b0d6f8ed9a11732a",
			  "arm64_sha256": "cd3dc9d182a4561b162f03c6986f4647bbb704f8e7e4872ee0611b1b9e86e1b0",
			  "repo": "registry.hub.docker.com/netdata/netdata",
			  "tag": "latest"
			},
			"vere": {
			  "amd64_sha256": "7dc0a1f97214101482d9c329a5108471bcf23fafea421e1ae2662c6c20377037",
			  "arm64_sha256": "1dbded539bd99cd789bfe5cbe11bb89baa598e213881389e8138da8fc0a27fa9",
			  "repo": "registry.hub.docker.com/nativeplanet/urbit",
			  "tag": "v2.11"
			},
			"webui": {
			  "amd64_sha256": "a2c683530d15b1095bcb3e97b1b6236461142e6e19774762991e30c817352797",
			  "arm64_sha256": "0b96f7a02efc0d5c754bd6d89df2b9f738a676c51b529258b99d5b84e30d1725",
			  "repo": "registry.hub.docker.com/nativeplanet/groundseg-webui",
			  "tag": "latest"
			},
			"wireguard": {
			  "amd64_sha256": "ae6f8e8cc1303bc9c0b5fa1b1ef4176c25a2c082e29bf8b554ce1196731e7db2",
			  "arm64_sha256": "403d741b1b5bcf5df1e48eab0af8038355fae3e29419ad5980428f9aebd1576c",
			  "repo": "registry.hub.docker.com/linuxserver/wireguard",
			  "tag": "latest"
			}
		  }
		}
	  }`
	VersionInfo structs.Version
)

func init() {
	if err := json.Unmarshal([]byte(DefaultVersionText), &VersionInfo); err != nil {
		errmsg := fmt.Sprintf("Error unmarshalling default version info: %v", err)
		logger.Error(errmsg)
	}
}
