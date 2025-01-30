from sandboxai import Sandbox
import logging
from textwrap import dedent
from swarm import Agent, Swarm

logging.basicConfig(level=logging.INFO)


def main():
    swarm_client = Swarm()

    with Sandbox(embedded=True, lazy_create=True) as box:
        agent = Agent(
            name="Git Repo Analyzer",
            instructions=dedent(
                """\
                Your job is to analyze a git repository and count the lines of code by language.
                Use the dedicated container environment that was provisioned for you to run shell python code and shell commands to accomplish your task.
                Act autonomously and do not ask any questions.
                """
            ),
            functions=[
                box.run_shell_command,
                box.run_ipython_cell,
            ],
            model="gpt-4o-mini",
        )

        response = swarm_client.run(
            agent=agent,
            messages=[
                {
                    "role": "user",
                    "content": "Analyze this repo: https://github.com/mattfeltonma/python-sample-web-app",
                }
            ],
            debug=True,
            model_override="gpt-4o-mini",
        )

        print(response.messages[-1]["content"])


if __name__ == "__main__":
    main()
