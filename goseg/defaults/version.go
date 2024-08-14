package defaults

import (
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"log/slog"
	"os"

	"go.uber.org/zap"
)

var (
	logger             = slog.New(slog.NewJSONHandler(os.Stdout, nil))
  DefaultVersionText = `
{
  "groundseg": {
    "canary": {
      "groundseg": {
        "amd64_sha256": "411f45433708c6755a354e702c8164a11e532ccb82d0d0af1f01b251e2789d1c",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.2.0_latest",
        "arm64_sha256": "84c8200e55b7737db75143a92fbfbc721042e7134d69f5860503408ae4bd3168",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.2.0_latest",
        "major": 2,
        "minor": 2,
        "patch": 0
      },
      "manual": {
        "amd64_sha256": "465a82af809481ce8c4861951be5d714a6e578e4330e6d7d7367fe1b170755a9",
        "arm64_sha256": "1110245032e88d627ce625ed25758b21a473aca38251354c1815ba4938f8d23e",
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
        "amd64_sha256": "2fdc4be94c1206344b2e969c292d7cc28e3ff751e00b32680527468ecfb7b7d9",
        "arm64_sha256": "None",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "canary"
      },
      "webui": {
        "amd64_sha256": "cc6ea93a53dcd50bef7be7077c41dc475943baee83343cece13884cb2a351308",
        "arm64_sha256": "eebd8b1041fc5216922d36760b884d6ee80cf42a4bf7afe3a3a397411ee2eb4b",
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
        "amd64_sha256": "6b3f9aedb6fdd409fb74b8ac6b0206d5662f076f0247f813e93742ef3ec82026",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.2.1-rc0_edge",
        "arm64_sha256": "8e5a1976a104ffadcfe100e7dbce98768c3f2e9c653233da98b324070e4f59fc",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.2.1-rc0_edge",
        "major": 2,
        "minor": 2,
        "patch": 1
      },
      "manual": {
        "amd64_sha256": "318a4a8922197493eefb73bfbd568728b5278f4146d2ba712929a032fd495092",
        "arm64_sha256": "1110245032e88d627ce625ed25758b21a473aca38251354c1815ba4938f8d23e",
        "repo": "registry.hub.docker.com/nativeplanet/groundseg-manual",
        "tag": "edge"
      },
      "minio": {
        "amd64_sha256": "6d6cf693fd70ca6e15709fa44d39b44f98fc5b58795697486a95ac1cc2ad9880",
        "arm64_sha256": "510eb939d4651d02806e696ff37c71902a17b8297b4a241670f7b59fd2eb4415",
        "repo": "registry.hub.docker.com/minio/minio",
        "tag": "latest"
      },
      "miniomc": {
        "amd64_sha256": "3455a7bae6058ea83f797a95c0e29a4daedff6f79b1f87a0ede429e0344734ab",
        "arm64_sha256": "599ceab1947ab8694e63d7c5e708e616d8dcc77cc26d4e9c36e20100efe84025",
        "repo": "registry.hub.docker.com/minio/mc",
        "tag": "latest"
      },
      "netdata": {
        "amd64_sha256": "4a3a8e1e79e31e3380a79493f53fdaba94d414c90954ed9a96690e9e6406bcf0",
        "arm64_sha256": "623840bab070e05cddadb197678598a58f2759d6685d4bcefb5881cfc5304f63",
        "repo": "registry.hub.docker.com/netdata/netdata",
        "tag": "latest"
      },
      "vere": {
        "amd64_sha256": "c968f7277bead15f8a29d1843fe98ee8515c00ed80650c05a58498d0d7b40656",
        "arm64_sha256": "23b6218a48575aa981568c622c1937af7291665cf27d6f57c45e87eabab06cf9",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "v3.0"
      },
      "webui": {
        "amd64_sha256": "cc6ea93a53dcd50bef7be7077c41dc475943baee83343cece13884cb2a351308",
        "arm64_sha256": "eebd8b1041fc5216922d36760b884d6ee80cf42a4bf7afe3a3a397411ee2eb4b",
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
        "amd64_sha256": "411f45433708c6755a354e702c8164a11e532ccb82d0d0af1f01b251e2789d1c",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.2.0_latest",
        "arm64_sha256": "84c8200e55b7737db75143a92fbfbc721042e7134d69f5860503408ae4bd3168",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.2.0_latest",
        "major": 2,
        "minor": 2,
        "patch": 0
      },
      "manual": {
        "amd64_sha256": "465a82af809481ce8c4861951be5d714a6e578e4330e6d7d7367fe1b170755a9",
        "arm64_sha256": "1110245032e88d627ce625ed25758b21a473aca38251354c1815ba4938f8d23e",
        "repo": "registry.hub.docker.com/nativeplanet/groundseg-manual",
        "tag": "latest"
      },
      "minio": {
        "amd64_sha256": "6d6cf693fd70ca6e15709fa44d39b44f98fc5b58795697486a95ac1cc2ad9880",
        "arm64_sha256": "510eb939d4651d02806e696ff37c71902a17b8297b4a241670f7b59fd2eb4415",
        "repo": "registry.hub.docker.com/minio/minio",
        "tag": "latest"
      },
      "miniomc": {
        "amd64_sha256": "3455a7bae6058ea83f797a95c0e29a4daedff6f79b1f87a0ede429e0344734ab",
        "arm64_sha256": "599ceab1947ab8694e63d7c5e708e616d8dcc77cc26d4e9c36e20100efe84025",
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
        "amd64_sha256": "164137948e7e6af3d7b7da9e0008adf1331f2f1e9de3c3384f5921a9b57a0c7c",
        "arm64_sha256": "727d43a00ace1612962abe2ea9c0462eb539a191bd3c7093b4567b8b9b163c92",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "v3.1"
      },
      "webui": {
        "amd64_sha256": "cc6ea93a53dcd50bef7be7077c41dc475943baee83343cece13884cb2a351308",
        "arm64_sha256": "eebd8b1041fc5216922d36760b884d6ee80cf42a4bf7afe3a3a397411ee2eb4b",
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
}
`
	VersionInfo structs.Version
)

func init() {
	if err := json.Unmarshal([]byte(DefaultVersionText), &VersionInfo); err != nil {
		zap.L().Error(fmt.Sprintf("Error unmarshalling default version info: %v", err))
	}
}
