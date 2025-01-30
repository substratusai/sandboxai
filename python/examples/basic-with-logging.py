from sandboxai import Sandbox
import logging

logging.basicConfig(level=logging.DEBUG)

with Sandbox(embedded=True) as box:
    print(box.run_ipython_cell("print('hi')").output)
