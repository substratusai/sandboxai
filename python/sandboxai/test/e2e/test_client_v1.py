import os
import pytest
import json

from sandboxai.api.v1 import (
    SandboxSpec,
    CreateSandboxRequest,
    RunIPythonCellRequest,
    RunShellCommandRequest,
)
from sandboxai.client.v1 import HttpClient


@pytest.fixture
def client():
    # Adjust the base_url as needed for your environment
    base_url = os.environ.get("SANDBOXAI_BASE_URL", "")
    return HttpClient(base_url=base_url)


def test_http_client_v1(client):
    """
    Run IPython tool end-to-end test for the Client.

    This test verifies that the client can create a sandbox, retrieve it,
    and run multiple IPython cells and shell command with the expected outputs.
    """

    with open(os.environ.get("TEST_SANDBOX_PATH"), "r") as f:
        # TODO: Load sbx (Sandbox() DataModel) from json
        data = json.load(f)
        spec = SandboxSpec(**data)

    assert spec.image == "placeholder-image", "Read unexpected sandbox image."
    spec.image = os.environ.get("BOX_IMAGE", "")

    space = "default"

    # Create a new sandbox (adjust the image as needed for your tests)
    sandbox = client.create_sandbox(space, CreateSandboxRequest(spec=spec))
    assert sandbox.name is not None, "No name returned for created sandbox."
    assert sandbox.name != "", "Empty name returned for created sandbox."

    # Ensure sandbox retrieval works
    retrieved = client.get_sandbox(space, sandbox.name)
    assert sandbox.model_dump() == retrieved.model_dump(), (
        "Created sandbox does not match retrieved sandbox."
    )

    # Read test cases from files.
    with open(os.environ.get("TEST_IPYTHON_CASES_PATH"), "r") as f:
        ipy_test_cases = json.load(f)

    with open(os.environ.get("TEST_SHELL_CASES_PATH"), "r") as f:
        shell_test_cases = json.load(f)

    # Make sure the cases have at least one test case.
    assert len(ipy_test_cases) > 0, "IPython cases empty."
    assert len(shell_test_cases) > 0, "Shell cases empty."

    try:
        for tc in ipy_test_cases:
            req = RunIPythonCellRequest(
                code=tc["code"], split_output=tc.get("split", False)
            )
            resp = client.run_ipython_cell(space, sandbox.name, req)

            # If we expect the output to contain a substring
            if "expected_output_contains" in tc:
                assert tc["expected_output_contains"] in (resp.output or "")
                # This check ensures there's no conflicting expectations
                assert "expected_output" not in tc, (
                    "Invalid assertion combo: both 'output' and 'output_contains' set."
                )
            else:
                # Must match exact output if 'expected_output' is defined
                expected_output = tc.get("expected_output", "")
                assert expected_output == (resp.output or "")

                if "expected_stdout" in tc:
                    assert tc["expected_stdout"] == (resp.stdout or "")
                if "expected_stderr" in tc:
                    assert tc["expected_stderr"] == (resp.stderr or "")

        for tc in shell_test_cases:
            req = RunShellCommandRequest(
                command=tc["command"], split_output=tc.get("split", False)
            )
            resp = client.run_shell_command(space, sandbox.name, req)

            if "expected_output" in tc:
                assert tc["expected_output"] == (resp.output or "")
            if "expected_stdout" in tc:
                assert tc["expected_stdout"] == (resp.stdout or "")
            if "expected_stderr" in tc:
                assert tc["expected_stderr"] == (resp.stderr or "")

    finally:
        client.delete_sandbox(space, sandbox.name)
