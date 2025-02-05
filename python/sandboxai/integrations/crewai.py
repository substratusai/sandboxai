from crewai.tools import BaseTool
from typing import Type
from pydantic import BaseModel, Field
from sandboxai import Sandbox


class RunIPythonCellArgs(BaseModel):
    cell: str = Field(..., description="The code to execute in the ipython cell.")


class RunIPythonCell(BaseTool):
    name: str = "Run"
    description: str = "Run python and shell commands in an ipython cell. Shell commands should be on a new line and start with a !."
    args_schema: Type[BaseModel] = RunIPythonCellArgs

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._sandbox = Sandbox(embedded=True)

    def __del__(self):
        self._sandbox.delete()

    def _run(self, cell: str) -> str:
        result = self._sandbox.run_ipython_cell(input=cell)
        return result.output


class RunShellCommandArgs(BaseModel):
    command: str = Field(..., description="The bash commands to execute.")


class RunShellCommand(BaseTool):
    name: str = "Run"
    description: str = "Run bash shell commands in a sandbox."
    args_schema: Type[BaseModel] = RunShellCommandArgs

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._sandbox = Sandbox(embedded=True)

    def __del__(self):
        self._sandbox.delete()

    def _run(self, command: str) -> str:
        result = self._sandbox.run_shell_command(command=command)
        return result.output
