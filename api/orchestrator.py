from time import sleep
import asyncio

class Orchestrator:
    def __init__(self, state):
        # GroundSeg Library
        # Loops
        # Docker
        # StarTram
        # Linux System

        print("orchestrator:__init__ Started")
        self.state = state
        print("orchestrator.py init")
        print("orchestrator:__init__ temp 10 second sleep")
        sleep(10)
        self.state['ready'] = True 

    def handle_request(self, action, websocket):
        # Process the received message
        return "DONE"
