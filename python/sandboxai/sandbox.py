from dataclasses import dataclass

from sandboxai.client.v1 import HttpClient
from sandboxai.api import v1 as v1Api
from sandboxai import embedded

from logging import getLogger

from threading import Lock

log = getLogger(__name__)

DEFAULT_IMAGE = "substratusai/sandboxai-box:v0.2.0"

# Prevent multiple Sandbox() instances from attempting to start the
# embedded server at the same time.
embedded_mutex = Lock()


@dataclass
class IPythonCellResult:
    output: str


@dataclass
class ShellCommandResult:
    output: str


class Sandbox:
    def __init__(
        self,
        base_url: str = "",
        embedded: bool = False,
        lazy_create: bool = False,
        space: str = "default",
        name: str = None,
        image: str = DEFAULT_IMAGE,
        env: dict = None,
    ):
        """
        Initialize a Sandbox instance.
        """
        self.space = space
        self.name = name
        self.image = image
        self.env = env
        if embedded:
            self.__launch_embdedded_server()
        else:
            if not base_url:
                raise ValueError("base_url or embedded must be specified")
            self.base_url = base_url

        self.client = HttpClient(self.base_url)

        if not lazy_create:
            self.create()

    def __enter__(self):
        """
        Enter the context manager.
        """
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        """
        Exit the context manager. Deletes the sandbox.
        """
        self.delete()
        return False  # Don't suppress any exceptions

    def create(self) -> None:
        created = self.client.create_sandbox(
            self.space,
            v1Api.CreateSandboxRequest(
                name=self.name, spec=v1Api.SandboxSpec(image=self.image, env=self.env)
            ),
        )
        self.name = created.name
        self.image = created.spec.image

    def delete(self) -> None:
        if self.name:
            self.client.delete_sandbox(self.space, self.name)
            self.name = ""
            self.image = ""

    def run_ipython_cell(self, input: str) -> IPythonCellResult:
        """
        Runs an ipython cell in the sandbox.
        """
        if not self.name:
            self.create()

        log.debug(f"Running ipython cell with input: {input}")
        result = self.client.run_ipython_cell(
            self.space,
            self.name,
            v1Api.RunIPythonCellRequest(code=input, split_output=False),
        )  # type: ignore
        log.debug(f"IPython cell returned the output: {result.output}")
        result = IPythonCellResult(output=result.output or "")
        return result

    def run_shell_command(self, command: str) -> ShellCommandResult:
        """
        Runs a shell command in the sandbox.
        """
        if not self.name:
            self.create()

        log.debug(f"Running shell command with input: {command}")
        result = self.client.run_shell_command(
            self.space,
            self.name,
            v1Api.RunShellCommandRequest(command=command, split_output=False),
        )  # type: ignore
        log.debug(f"Shell command returned the output: {result.output}")
        result = ShellCommandResult(output=result.output or "")
        return result

    def __launch_embdedded_server(self):
        global embedded_mutex
        with embedded_mutex:
            if not embedded.is_running():
                log.info("Starting embedded server...")
                embedded.start_server()
                self.base_url = embedded.get_base_url()
            else:
                base_url = embedded.get_base_url()
                log.info(f"Embedded server is already running at {base_url}.")
                self.base_url = base_url
