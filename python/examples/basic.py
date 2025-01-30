from sandboxai import Sandbox

with Sandbox(embedded=True) as box:
    print(box.run_ipython_cell("print('hi')").output)
