'''
import json

class JSONConstructor:
    def __init__(self, initial_data=None):
        if initial_data is None:
            self.data = {}
        elif isinstance(initial_data, dict):
            self.data = initial_data
        else:
            raise ValueError("Initial data must be a dictionary or None.")

    def add_element(self, key, value):
        self.data[key] = value

    def update_element(self, key, value):
        if key in self.data:
            self.data[key] = value
        else:
            raise KeyError(f"Key '{key}' not found in the JSON data.")

    def remove_element(self, key):
        if key in self.data:
            del self.data[key]
        else:
            raise KeyError(f"Key '{key}' not found in the JSON data.")

    def to_string(self, indent=None, sort_keys=False):
        return json.dumps(self.data, indent=indent, sort_keys=sort_keys)

# Usage example
#json_constructor = JSONConstructor({"name": "John", "age": 30})
#json_constructor.add_element("city", "New York")
#json_constructor.update_element("age", 31)
#json_constructor.remove_element("city")
#json_string = json_constructor.to_string(indent=4, sort_keys=True)

#print(json_string)
'''
