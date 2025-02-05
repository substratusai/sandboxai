from crewai.tools import BaseTool
from typing import Type
from pydantic import BaseModel, Field
from sandboxai import Sandbox


class SandboxIPythonToolArgs(BaseModel):
    code: str = Field(..., description="The code to execute in the ipython cell.")


class SandboxIPythonTool(BaseTool):
    name: str = "Run Python code"
    description: str = "Run python code and shell commands in an ipython cell. Shell commands should be on a new line and start with a '!'."
    args_schema: Type[BaseModel] = SandboxIPythonToolArgs

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        # Note that the sandbox only shuts down once the Python program exits.
        self._sandbox = Sandbox(embedded=True)

    def _run(self, code: str) -> str:
        result = self._sandbox.run_ipython_cell(code=code)
        return result.output


class SandboxShellToolArgs(BaseModel):
    command: str = Field(..., description="The bash commands to execute.")


class SandboxShellTool(BaseTool):
    name: str = "Run shell command"
    description: str = "Run bash shell commands in a sandbox."
    args_schema: Type[BaseModel] = SandboxShellToolArgs

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        # Note that the sandbox only shuts down once the Python program exits.
        self._sandbox = Sandbox(embedded=True)

    def _run(self, command: str) -> str:
        result = self._sandbox.run_shell_command(command=command)
        return result.output
