import shutil
import subprocess
import json
import os
import atexit
import threading
import uuid

from logging import getLogger

log = getLogger(__name__)

__process = None
__running = False
__base_url = None
__logging_thread = None


def _stream_to_logger(pipe):
    """
    Helper function running in a thread.
    Reads lines from a subprocess pipe and logs them.
    """
    with pipe:
        for line in iter(pipe.readline, ""):
            # Remove trailing newline to avoid double spacing in logs
            line = line.rstrip()
            if line:  # Avoid empty lines
                log.debug(f"Server: {line}")


def start_server():
    """
    Launches a local SandboxAI server.
    """

    global __process
    global __base_url
    global __logging_thread
    global __running

    if not shutil.which("docker"):
        raise RuntimeError("docker not found on the system.")

    sandboxaid_path = os.path.join(os.path.dirname(__file__), "bin", "sandboxaid")
    if not os.path.isfile(sandboxaid_path):
        raise RuntimeError(f"Included sandboxaid not found at: {sandboxaid_path}")

    process_env = os.environ.copy()
    # Auto-select a free port.
    process_env["SANDBOXAID_PORT"] = "0"
    # Scope the management to this embedded instance.
    process_env["SANDBOXAID_SCOPE"] = str(uuid.uuid4())
    # When the server is stopped, delete all managed sandboxes.
    process_env["SANDBOXAID_DELETE_ON_SHUTDOWN"] = "true"

    # Launch the sandboxaid binary in the background
    __process = subprocess.Popen(
        [sandboxaid_path],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        env=process_env,
        # Ensure that the server does not receive interrupt signals at the same time
        # as the python program. This is important to allow for all sandbox deletion
        # requests to be processed before the server stops accepting new requests.
        # See the stop_server() function for the shutdown flow (registered via atexit).
        preexec_fn=os.setsid,
    )
    __running = True

    # Spawn a thread to read stderr and log its contents
    __logging_thread = threading.Thread(
        target=_stream_to_logger,
        args=(__process.stderr,),
        daemon=True,
    )
    __logging_thread.start()

    # Check if the process started successfully
    if __process.poll() is None:
        log.info("Sandboxd started successfully with PID: %d", __process.pid)
        # NOTE: Python __exit__ functions should complete before atexit functions run.
        # This allows the sandboxes started using `with:` statements to properly clean
        # themselves up by issuing DELETE requests to the embedded server.
        atexit.register(stop_server)
    else:
        raise RuntimeError("Failed to launch sandboxaid")

    if __process.stdout is None:
        raise RuntimeError("Failed to capture stdout")

    # Read the auto-selected port.
    first_line = __process.stdout.readline().strip()
    try:
        server_info = json.loads(first_line)
        port = server_info.get("port")
        __base_url = f"http://localhost:{port}/v1"
    except json.JSONDecodeError as e:
        __process.terminate()
        raise json.JSONDecodeError(
            f"Failed to decode first line as JSON: {first_line}", e.doc, e.pos
        ) from e


def stop_server():
    global __process
    global __running
    if __process:
        try:
            log.info("Terminating embedded server")
            __process.terminate()
            __running = False
            log.info("Waiting for embedded server to stop")
            __process.wait(timeout=30)
            log.info("Embedded server stopped")
            if __logging_thread:
                # Explicitly wait for the logging thread to stop as well, in case it's still running.
                # This is necessary to ensure all logs are written. The logging thread is running in
                # daemon mode because shutdown is triggered "atexit" which is triggered only AFTER all
                # threads stop.
                log.info("Waiting for logging thread to stop")
                __logging_thread.join(timeout=10)
                log.debug("Embedded logging thread stopped")
        except OSError:
            pass  # process may already be gone
    __process = None


def is_running():
    global __running
    return __running


def get_base_url():
    global __base_url
    return __base_url
