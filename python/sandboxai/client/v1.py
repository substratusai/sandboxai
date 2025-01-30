import time
import requests

from sandboxai.api.v1 import (
    Sandbox,
    CreateSandboxRequest,
    RunIPythonCellRequest,
    RunIPythonCellResult,
    RunShellCommandRequest,
    RunShellCommandResult,
)


def _validate_response(response: requests.Response, expected_status: int) -> None:
    """
    Validates that the response's status code matches the expected status.
    Raises an exception if there's a mismatch.
    """
    if response.status_code != expected_status:
        raise RuntimeError(
            f"Expected status {expected_status}, got {response.status_code}: {response.text}"
        )


class SandboxNotFoundError(Exception):
    def __init__(self, message: str):
        super().__init__(message)


class HttpClient:
    """
    A Python client for interacting with the SandboxAI API, using typed models from v1.py.
    Provides CRUD operations on sandbox resources and runs IPython cells.
    """

    def __init__(self, base_url: str) -> None:
        """
        Initialize the Python client.

        Args:
            base_url (str): The base URL for the SandboxAI API
                            (for example, "http://localhost:5000/v1").
        """
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def check_health(self) -> bool:
        """
        Checks if the sandbox service is running by verifying the health endpoint.

        Returns:
            bool: True if the service is reachable, False otherwise
        """
        endpoint = f"{self.base_url}/healthz"
        try:
            response = self.session.get(endpoint)
            return response.status_code == 200
        except requests.RequestException:
            return False

    def wait_until_healthy(self, timeout: int = 10) -> None:
        """
        Waits until the sandbox service is running for a specified timeout.
        """
        start_time = time.time()
        while time.time() - start_time < timeout:
            if self.check_health():
                return
            time.sleep(1)
        raise TimeoutError(
            "Sandbox service did not start within the specified timeout."
        )

    def create_sandbox(self, space: str, req: CreateSandboxRequest) -> Sandbox:
        """
        Create a new sandbox.

        Args:
            req (CreateSandboxRequest): Sandbox to create.

        Returns:
            Sandbox: The newly created sandbox, as returned by the API.
        """
        endpoint = f"{self.base_url}/spaces/{space}/sandboxes"
        response = self.session.post(endpoint, json=req.model_dump())
        _validate_response(response, 201)
        return Sandbox.model_validate(response.json())

    def get_sandbox(self, space: str, name: str) -> Sandbox:
        """
        Retrieve an existing sandbox by its ID.

        Args:
            space (str): Space where the sandbox lives.
            name (str): Name of the sandbox.

        Returns:
            Sandbox: The retrieved sandbox.

        Raises:
            SandboxNotFoundError: If the sandbox with the given ID does not exist.
        """
        endpoint = f"{self.base_url}/spaces/{space}/sandboxes/{name}"
        response = self.session.get(endpoint)
        if response.status_code == 404:
            raise SandboxNotFoundError(
                f"Sandbox with name '{name}' not found in space '{space}'."
            )
        _validate_response(response, 200)
        return Sandbox.model_validate(response.json())

    def delete_sandbox(self, space: str, name: str) -> None:
        """
        Delete an existing sandbox.

        Args:
            space (str): Space where the sandbox lives.
            name (str): Name of the sandbox.
        """
        endpoint = f"{self.base_url}/spaces/{space}/sandboxes/{name}"
        response = self.session.delete(endpoint)
        _validate_response(response, 204)

    def run_ipython_cell(
        self, space: str, name: str, request: RunIPythonCellRequest
    ) -> RunIPythonCellResult:
        """
        Run an IPython cell in the specified sandbox.

        Args:
            space (str): Space where the sandbox lives.
            name (str): Name of the sandbox.
            request (RunIPythonCellRequest): The cell execution request details.

        Returns:
            RunIPythonCellResult: The result of running the cell.
        """
        endpoint = (
            f"{self.base_url}/spaces/{space}/sandboxes/{name}/tools:run_ipython_cell"
        )
        response = self.session.post(endpoint, json=request.model_dump())
        _validate_response(response, 200)
        return RunIPythonCellResult.model_validate(response.json())

    def run_shell_command(
        self, space: str, name: str, request: RunShellCommandRequest
    ) -> RunShellCommandResult:
        """
        Run a shell command in the specified sandbox.

        Args:
            space (str): Space where the sandbox lives.
            name (str): Name of the sandbox.
            request (RunShellCommandRequest): The shell command execution request details.

        Returns:
            RunShellCommandResult: The result of running the shell command.
        """
        endpoint = (
            f"{self.base_url}/spaces/{space}/sandboxes/{name}/tools:run_shell_command"
        )
        response = self.session.post(endpoint, json=request.model_dump())
        _validate_response(response, 200)
        return RunShellCommandResult.model_validate(response.json())
