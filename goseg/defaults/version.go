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
        "amd64_sha256": "dae50918c6c67da3ac738fa0bede05f14d83dc31ee8f17ca4311c542dd5550d8",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_edge_v2.4.11-rc2",
        "arm64_sha256": "6144e7834d422891e470bd58faf7b2ada08c1ca4757dc4ae5731d17a88db11da",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_edge_v2.4.11-rc2",
        "slsa_url": "https://files.native.computer/bin/groundseg_latest_v2.4.13.jsonl",
        "major": 2,
        "minor": 4,
        "patch": 11
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
        "amd64_sha256": "1e75a40f2ba19939c1cc538a381111f79fe541ee1732712fd64027596dabf68d",
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
      },
      "rustfs": {
        "amd64_sha256": "35c522d3926bfb3129eb9a9eb8c3431367d15366c63d973987a827a213ee7954",
        "arm64_sha256": "62117ed0cbaf1326c0710530a7a6440d2976b92d29cb9ed8c43a339a340f5e87",
        "repo": "registry.hub.docker.com/rustfs/rustfs",
        "tag": "latest"
      }
    },
    "edge": {
      "groundseg": {
        "amd64_sha256": "306aab3a754a88caf5ad49c841a07cec829a89ec7551e06a6f11f67d68d49347",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_edge_v2.5.0-rc0",
        "arm64_sha256": "c73fc6c94adedf70fe3505ae902055ae19bd920f2af506ae11d1e75be826a2d5",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_edge_v2.5.0-rc0",
        "slsa_url": "https://files.native.computer/bin/groundseg_edge_v2.5.0-rc0.jsonl",
        "major": 2,
        "minor": 5,
        "patch": 0
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
        "repo": "registry.hub.docker.com/rustfs/rustfs",
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
        "amd64_sha256": "a4af757cd710226be794c32fcd83589db6da19bedba8b90d60455ab5c8ff20a1",
        "arm64_sha256": "5183d3fa645015e204bb3e075e50484c7dc3dd37b9f3b191f2454ef2b1d35d1e",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "v4.3"
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
      },
      "rustfs": {
        "amd64_sha256": "35c522d3926bfb3129eb9a9eb8c3431367d15366c63d973987a827a213ee7954",
        "arm64_sha256": "62117ed0cbaf1326c0710530a7a6440d2976b92d29cb9ed8c43a339a340f5e87",
        "repo": "registry.hub.docker.com/rustfs/rustfs",
        "tag": "latest"
      }
    },
    "latest": {
      "groundseg": {
        "amd64_sha256": "55a3c041acb33f1a2da8e1902d3243f0bf650a16a1ad3cd309a7f0da338401c0",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_latest_v2.4.13",
        "arm64_sha256": "dcb2899ed59245416c72b57daea25b9d43ce0548752a0f8bdac8918fc0fc4c81",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_latest_v2.4.13",
        "slsa_url": "https://files.native.computer/bin/groundseg_latest_v2.4.13.jsonl",
        "major": 2,
        "minor": 4,
        "patch": 13
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
        "amd64_sha256": "7194cdf9df489ec68155a7d801c4444cd76857702d0cefc59de8dc04a4c9cc13",
        "arm64_sha256": "f96275f3c79dd9d7154f077890c52edb4aa808386d4472dc77b5bb7325a2135f",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "v4.3"
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
      },
      "rustfs": {
        "amd64_sha256": "35c522d3926bfb3129eb9a9eb8c3431367d15366c63d973987a827a213ee7954",
        "arm64_sha256": "62117ed0cbaf1326c0710530a7a6440d2976b92d29cb9ed8c43a339a340f5e87",
        "repo": "registry.hub.docker.com/rustfs/rustfs",
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
