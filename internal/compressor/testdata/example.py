# This is a Python comment
import os
from sys import argv

class MyClass:
    """
    A simple class
    """
    def __init__(self, name):
        self.name = name

    def greet(self, message):
        """Greets the person."""
        # Method body
        print(f"{message}, {self.name}!")
        if len(self.name) > 3:
            print("Long name")
        return True

def my_function(x, y):
    # Function body
    result = x + y
    print(f"Result is {result}")
    return result

if __name__ == "__main__":
    c = MyClass("Test")
    c.greet("Hello")
    my_function(1, 2)
