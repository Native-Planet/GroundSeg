from graphene import ObjectType, Field, String, Boolean, List

class Pier(ObjectType):
    # General (requested on home page)
    name = String(patp=String(default_value="all"))
    running = Boolean() #urbit.isRunning();
    code = String()

    # Pier details
    urbit_url = String()
    minio_url = String()
    minio_reg = Boolean()
    remote = Boolean()

    # Advanced options
    pier_logs = String()
    minio_adv = String()
    pier_admin = String()


class Query(ObjectType):
    all_piers = List(Pier)
    pier = Field(Pier)
    system = String()

    def resolve_piers(root, info):

        return [
                {
                    "name":"nallux-dozryl",
                    "running":True,
                    "code": "nidpel-ripnus-niswex-bicted",
                    "urbit_url":"https://darmud-hasnep-nallux-dozryl.startram.io",
                    "minio_url":"https://console.s3.darmud-hasnep-nallux-dozryl.startram.io",
                    "minio_reg":True,
                    "remote":True,
                    "pier_logs":"sum logssssss",
                    "minio_adv": "linked/exported bucket",
                    "pier_admin": "restarted/down has been shut"
                    },
                {
                    "name":"donpub-dozryl",
                    "running":True,
                    "code": "nidpzl-ripnus-niswex-bicted",
                    "urbit_url":"https://darmuz-hasnep-nallux-dozryl.startram.io",
                    "minio_url":"https://consoze.s3.darmud-hasnep-nallux-dozryl.startram.io",
                    "minio_reg":True,
                    "remote":True,
                    "pier_logs":"sum logsssssz",
                    "minio_adv": "linked/expozted bucket",
                    "pier_admin": "restarted/zown has been shut"
                    },
                {
                    "name":"nallux-dozryl",
                    "running":True,
                    "code": "nidpel-ripnus-niswex-bicted",
                    "urbit_url":"https://darmud-hasnep-nallux-dozryl.startram.io",
                    "minio_url":"https://console.s3.darmud-hasnep-nallux-dozryl.startram.io",
                    "minio_reg":True,
                    "remote":True,
                    "pier_logs":"sum logssssss",
                    "minio_adv": "linked/exported bucket",
                    "pier_admin": "restarted/down has been shut"
                    },
                {
                    "name":"donpub-dozryl",
                    "running":True,
                    "code": "nidpzl-ripnus-niswex-bicted",
                    "urbit_url":"https://darmuz-hasnep-nallux-dozryl.startram.io",
                    "minio_url":"https://consoze.s3.darmud-hasnep-nallux-dozryl.startram.io",
                    "minio_reg":True,
                    "remote":True,
                    "pier_logs":"sum logsssssz",
                    "minio_adv": "linked/expozted bucket",
                    "pier_admin": "restarted/zown has been shut"
                    }
                ]

    def resolve_system(root, info):
        return 'settings dump'
