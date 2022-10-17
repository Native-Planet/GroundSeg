from graphene import ObjectType, Field, String, Boolean, List

class Pier(ObjectType):
    # General (requested on home page)
    name = String()
    running = Boolean() #urbit.isRunning();
    code = String()

    # Pier details
    urbit_url = String()
    minio_url = String()
    minio_reg = Boolean()
    remote = String()

    # Advanced options
    pier_logs = String()
    minio_adv = String()
    pier_admin = String()

class Query(ObjectType):
    piers = List(Pier)
    system = String()

    def resolve_piers(root, info):
        return [{
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
            }]

    def resolve_system(root, info):
        return 'settings dump'
