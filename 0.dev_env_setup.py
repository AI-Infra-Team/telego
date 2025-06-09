import sys
import os


def main():
    # install pyscript-util
    pythoncmd = "python3"
    final_notify = ""
    if sys.platform == "win32":
        pythoncmd = "python"

    # install the latest version of pyscript-util
    os.system(f"{pythoncmd} -m pip install pyscript-util --upgrade")

    # install pyscript-util
    import pyscript_util
    from pyscript_util import stage

    with stage("chdir_to_cur_file"):
        pyscript_util.chdir_to_cur_file()

    with stage("setup_npm"):
        pyscript_util.setup_npm()

    with stage("install ruff"):
        # check ruff is installed
        if pyscript_util.run_cmd("ruff --version") != 0:
            pyscript_util.run_cmd_sure(
                "curl -LsSf https://astral.sh/ruff/install.sh | sh"
            )
        final_notify += "remember to install vscode/cursor extension for ruff\n"

    with stage("update cursorrules with pyscript-util usage"):
        pyscript_util.add_usage_to_cursorrule(".cursorrules")

    with stage("conclusion"):
        print(final_notify)
        print("done")


if __name__ == "__main__":
    main()
