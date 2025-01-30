from fastapi import FastAPI, HTTPException
from IPython.core.interactiveshell import InteractiveShell
from contextlib import redirect_stdout, redirect_stderr
import io
import subprocess

from sandboxai.api.v1 import (
    RunIPythonCellRequest,
    RunIPythonCellResult,
    RunShellCommandRequest,
    RunShellCommandResult,
)

# Initialize FastAPI app
app = FastAPI(
    title="Box Daemon",
    version="1.0",
    description="The server that runs python code and shell commands in a SandboxAI environment.",
)

# Initialize IPython shell
ipy = InteractiveShell.instance()


@app.get(
    "/healthz",
    summary="Check the health of the API",
    response_model=None,
)
async def healthz():
    return {"status": "OK"}


@app.post(
    "/tools:run_ipython_cell",
    response_model=RunIPythonCellResult,
    summary="Invoke a cell in a stateful IPython (Jupyter) kernel",
)
async def run_ipython_cell(request: RunIPythonCellRequest):
    """
    Execute code in an IPython kernel and return the results.

    Args:
        request: The cell execution request containing the code to run

    Returns:
        The execution results including output, stdout, and stderr
    """
    try:
        if request.split_output:
            # Capture stdout and stderr separately
            stdout_buf = io.StringIO()
            stderr_buf = io.StringIO()

            with redirect_stdout(stdout_buf), redirect_stderr(stderr_buf):
                ipy.run_cell(request.code)

            return RunIPythonCellResult(
                stdout=stdout_buf.getvalue(), stderr=stderr_buf.getvalue()
            )
        else:
            # Capture combined output
            output_buf = io.StringIO()
            with redirect_stdout(output_buf), redirect_stderr(output_buf):
                ipy.run_cell(request.code)

            return RunIPythonCellResult(output=output_buf.getvalue())

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.post(
    "/tools:run_shell_command",
    response_model=RunShellCommandResult,
    summary="Invoke a shell command.",
)
async def run_shell_command(request: RunShellCommandRequest):
    """
    Execute a shell command and return the results.
    """
    try:
        output, stdout, stderr = None, None, None
        if request.split_output:
            # Split output mode: capture stdout and stderr separately
            result = subprocess.run(
                request.command,
                shell=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
            )
            stdout = result.stdout
            stderr = result.stderr
        else:
            # Combined output mode: redirect stderr to stdout
            result = subprocess.run(
                request.command,
                shell=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,  # Redirect stderr to stdout
                text=True,
            )
            output = result.stdout

        return RunShellCommandResult(
            output=output, stdout=stdout, stderr=stderr, return_code=result.returncode
        )

    except Exception as e:
        raise HTTPException(
            status_code=500, detail=f"Failed to execute shell command: {str(e)}"
        )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
