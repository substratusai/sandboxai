from sandboxai import Sandbox

with Sandbox(embedded=True, env={"FOO": "bar"}) as box:
    print(box.run_ipython_cell("! echo $FOO").output)
    print(box.run_shell_command("echo $FOO").output)
