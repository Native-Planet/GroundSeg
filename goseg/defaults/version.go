package defaults

import (
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"log/slog"
	"os"
)

var (
	logger             = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	DefaultVersionText = `
{
  "groundseg": {
    "canary": {
      "groundseg": {
        "amd64_sha256": "a29c5ecb84958f888c863e52c3d7fe9b3edfe0b4820190a03c105142f9ad131b",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.0.10_latest",
        "arm64_sha256": "dc6f29af0f33a184ffb18366d97a9efaa986fb562189dd47786bac3e29e047b2",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.0.10_latest",
        "major": 2,
        "minor": 0,
        "patch": 10
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
        "amd64_sha256": "2ef91ad9f76a7631afef89bc07875921b5ff0207410b5973824ab913c093b543",
        "arm64_sha256": "None",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "edge"
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
        "amd64_sha256": "c7520cbe5a2bafa24e48781bd8c0f49ee12142c9a8ad9c03dc46b0984d73ee4c",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.0.11-rc4_edge",
        "arm64_sha256": "155a2783b0e8b5b93bf16023a196734547573b7e700d21bb0b78e0c687aca486",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.0.11-rc4_edge",
        "major": 2,
        "minor": 0,
        "patch": 11
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
        "amd64_sha256": "81fb791042b29eb94c8c10cd8a9f6efb7032c3d7404ead471f25d4c1a63ebbb1",
        "arm64_sha256": "11ba34a3a0dc4e118280ed8a468d9f2289af82da1d9febb83e3b74de5bf3134f",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "v2.12"
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
        "amd64_sha256": "a29c5ecb84958f888c863e52c3d7fe9b3edfe0b4820190a03c105142f9ad131b",
        "amd64_url": "https://files.native.computer/bin/groundseg_amd64_v2.0.10_latest",
        "arm64_sha256": "dc6f29af0f33a184ffb18366d97a9efaa986fb562189dd47786bac3e29e047b2",
        "arm64_url": "https://files.native.computer/bin/groundseg_arm64_v2.0.10_latest",
        "major": 2,
        "minor": 0,
        "patch": 10
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
        "amd64_sha256": "81fb791042b29eb94c8c10cd8a9f6efb7032c3d7404ead471f25d4c1a63ebbb1",
        "arm64_sha256": "11ba34a3a0dc4e118280ed8a468d9f2289af82da1d9febb83e3b74de5bf3134f",
        "repo": "registry.hub.docker.com/nativeplanet/urbit",
        "tag": "v2.12"
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
		errmsg := fmt.Sprintf("Error unmarshalling default version info: %v", err)
		logger.Error(errmsg)
	}
}
