class Orchestrator:
    def __init__(self, state):
        self.state = state
        self.state['ready']['orchestrator'] = True 
