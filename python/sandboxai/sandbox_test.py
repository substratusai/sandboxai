import os
import pytest
from sandboxai import Sandbox
from sandboxai.sandbox import DEFAULT_IMAGE


@pytest.fixture
def box_image():
    return os.environ.get("BOX_IMAGE", DEFAULT_IMAGE)


@pytest.fixture
def base_url():
    return os.environ.get("SANDBOXAI_BASE_URL")


@pytest.fixture
def sandbox(box_image):
    with Sandbox(embedded=True, image=box_image) as sb:
        yield sb


def test_run_ipython_cell(sandbox):
    result = sandbox.run_ipython_cell("print(123)")
    assert "123" in result.output


def test_run_ipython_cell_error(sandbox):
    result = sandbox.run_ipython_cell("foo")
    assert "name 'foo' is not defined" in result.output


def test_run_shell_command_error(sandbox):
    result = sandbox.run_shell_command(">&2 echo 'error'")
    assert "error" in result.output


def test_sandbox_delete(sandbox):
    sandbox.delete()
    assert sandbox.name == ""
    assert sandbox.image == ""


def test_sandbox_lazy_create(box_image):
    try:
        sb = Sandbox(embedded=True, lazy_create=True, image=box_image)
        assert sb.name is None
        sb.create()
        assert sb.name != ""
    finally:
        sb.delete()


def test_with_specified_name(box_image):
    try:
        sb = Sandbox(embedded=True, lazy_create=True, image=box_image, name="test-name")
        assert sb.name == "test-name"
        sb.create()
        assert sb.name == "test-name"
    finally:
        sb.delete()


# These tests pass locally but fail on github actions.
# def test_sandbox_embedded_server():
#     with Sandbox(embedded=True) as sb:
#         assert sb.id != ""
#         assert isinstance(sb, Sandbox)
#
# def test_sandbox_embedded_server_existing():
#     with Sandbox(embedded=True) as sb1:
#         with Sandbox(embedded=True) as sb2:
#             assert sb1.base_url == sb2.base_url
# def test_sandbox_with_base_url(base_url):
#     if not base_url:
#         pytest.skip("SANDBOXAI_BASE_URL is not set")
#     with Sandbox(base_url=base_url) as sb:
#         assert sb.base_url == base_url
#         result = sb.run_ipython_cell("print('hi')").output
#         assert result == "hi\n"
