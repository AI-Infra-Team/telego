import os

def os_system(cmd):
    print(f"Running command: {cmd}")
    os.system(cmd)

def os_system_sure(cmd):
    print(f"Running command: {cmd}")
    os.system(cmd)
    if os.system(cmd) != 0:
        print(f"Command failed: {cmd}")
        exit(1)

def get_target_bin():
    # get cursys
    cursys = ""
    if os.name == "nt":
        cursys = "windows"
    else:
        cursys = "linux"

    # get curarch
    curarch = ""
    if os.uname().machine == "x86_64":
        curarch = "amd64"
    else:
        curarch = "arm64"

    targetbin=""
    if os.path.exists(f"./dist/telego_{cursys}_{curarch}"):
        targetbin = f"./dist/telego_{cursys}_{curarch}"
    else:
        targetbin = f"./dist/telego_{cursys}_{curarch}.exe"
    return targetbin

os_system_sure("python3 1.build.py")

# copy targetbin to /usr/bin
sudo=""
if os.geteuid() != 0:
    sudo="sudo "
os_system_sure(f"{sudo}cp {get_target_bin()} /usr/bin/telego")

# set NO_UPGRADE env to avoid auto upgrade
print("\nmaybe u r developing and testing")
print("set NO_UPGRADE env to avoid auto upgrade")
print("export NO_UPGRADE=true")