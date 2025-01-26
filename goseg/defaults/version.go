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
        "amd64_sha256": "9cf7b4ea6a81d4dffd5b0e379732d144f617ea1d74c0aaba0a056e17f4213c9a",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.4.0_latest",
        "arm64_sha256": "e0b240eb6fa289aa6f1e5e87cb1fc27820dc23c9c0eefaec3be126185a743f85",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.4.0_latest",
        "major": 2,
        "minor": 4,
        "patch": 0
      },
      "manual": {
        "amd64_sha256": "e59d3e83cffb7be6c56b58624caa095cb8ec4075ecff7962db510d891696ca86",
        "arm64_sha256": "41190934ca17ee553afa75bd286b539db3f8af066b8189b4646872c56b6bea63",
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
        "amd64_sha256": "a288be1742505b7dd0eadefbe8c379a4696031418384aa522c50f224bfdc497c",
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
        "amd64_sha256": "78a27a62ba8166b58e67148114c240c0207502e6e427ccccd60be2c6da6efc30",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.4.1-rc2_edge",
        "arm64_sha256": "18dbc71846ba5e09a992dafa504326237205cfddcfd019c3021d98002c1cfcf8",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.4.1-rc2_edge",
        "major": 2,
        "minor": 4,
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
        "amd64_sha256": "3da9d1419abbb0cd42f9c8f53a38a5b91baa00f7b4fc4dfb7fa65a3ade47d611",
        "arm64_sha256": "7e2eb9c2a8472a6c396a82f5fc7d629546a8f5d469e7e40bf1d87a8a05ba59d8",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "v3.1"
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
        "amd64_sha256": "9cf7b4ea6a81d4dffd5b0e379732d144f617ea1d74c0aaba0a056e17f4213c9a",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.4.0_latest",
        "arm64_sha256": "e0b240eb6fa289aa6f1e5e87cb1fc27820dc23c9c0eefaec3be126185a743f85",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.4.0_latest",
        "major": 2,
        "minor": 4,
        "patch": 0
      },
      "manual": {
        "amd64_sha256": "e59d3e83cffb7be6c56b58624caa095cb8ec4075ecff7962db510d891696ca86",
        "arm64_sha256": "41190934ca17ee553afa75bd286b539db3f8af066b8189b4646872c56b6bea63",
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
