import os
import hashlib
import subprocess
import yaml
import random

curfdir=os.path.dirname(os.path.abspath(__file__))
os.chdir(curfdir)
with open("compile_conf.yml",encoding="utf-8") as f:
    conf=yaml.safe_load(f)
main_node_ip=conf["main_node_ip"]
main_node_user=conf["main_node_user"]

def cmd_color(string,color):
    color_dict={"red":31,"green":32,"yellow":33,"blue":34,"magenta":35,"cyan":36,"white":37}
    return f"\033[{color_dict[color]}m{string}\033[0m"

def os_system_sure(command):
    BEFORE_RUN_TITLE=cmd_color("执行命令：","blue")
    RUN_FAIL_TITLE=cmd_color(">","blue")+"\n"+cmd_color("命令执行失败：","red")
    RUN_SUCCESS_TITLE=cmd_color(">","blue")+"\n"+cmd_color("命令执行成功：","green")
    print(f"{BEFORE_RUN_TITLE}{command}")
    code=os.system(command)
    # result, code = run_command2(command,allow_fail=True)
    # code=os.system(command)
    if code != 0:
        print(f"{RUN_FAIL_TITLE}{command}")
        exit(1)
    print(f"{RUN_SUCCESS_TITLE}{command}\n")
def os_system(command):
    BEFORE_RUN_TITLE=cmd_color("执行命令：","blue")
    RUN_FAIL_TITLE=cmd_color("\n命令执行失败：","red")
    RUN_SUCCESS_TITLE=cmd_color("\n命令执行成功：","green")
    print(f"{BEFORE_RUN_TITLE}{command}")
    code=os.system(command)
    # result =run_command2(command,allow_fail=True)
    if code != 0:
        print(f"{RUN_FAIL_TITLE}{command}")
    else:
        print(f"{RUN_SUCCESS_TITLE}{command}\n")
    return code

# 启动时检查是否为docker ubuntu18.04
def check_ubuntu_version():
    os_release_file = "/etc/os-release"
    
    if not os.path.exists(os_release_file):
        print("无法找到 /etc/os-release 文件，可能不是基于 Linux 的系统。")
        return False
    try:
        with open(os_release_file, 'r') as f:
            lines = f.readlines()

        os_info = {}
        for line in lines:
            key, _, value = line.strip().partition("=")
            os_info[key] = value.strip('"')

        if os_info.get("NAME") == "Ubuntu" and os_info.get("VERSION_ID") == "18.04":
            print("当前系统是 Ubuntu 18.04。")
            return True
        else:
            print(f"当前系统不是 Ubuntu 18.04，而是 {os_info.get('NAME')} {os_info.get('VERSION_ID', '未知版本')}。")
            return False
    except:
        print("无法打开 /etc/os-release 文件。")
        return False

if not check_ubuntu_version():
    os.system("python3 gen_menu.py")
    
    print("将启动 ubuntu 18.04进行编译")
    os.system("mkdir -p build")

    def _recover_proxy():
        pass
    recover_proxy=_recover_proxy

    # PROXY_ENV=""
    if os.environ.get("http_proxy"):  
        PROXY_ADDR=os.environ['http_proxy']
        if PROXY_ADDR.find("127.0.0.1")!=-1: 
            del os.environ['http_proxy']
            del os.environ['https_proxy']
            def r():
                os.environ['http_proxy']=PROXY_ADDR
                os.environ['https_proxy']=PROXY_ADDR
            recover_proxy=r
            
    # 创建dockerfile
    with open("build/Dockerfile", "w") as f:
        f.write(f"""
FROM ubuntu:18.04
                
RUN apt-get update
RUN apt-get install -y git
RUN apt-get install -y curl 
RUN apt-get install -y wget 
RUN apt-get install -y python3 
RUN apt-get install -y python3-yaml 
RUN apt-get clean

RUN wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz
RUN rm -rf /usr/local/go \
    && tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz \
    && rm -rf go1.23.2.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${{PATH}}"
ENV PYTHONIOENCODING=utf-8

CMD ["bash"]
""")
    
    os.chdir("build")
    flist=os.listdir("../")
    blacklist=[".cache",".git",".dist","build","telego_prjs","teleyard-template"]
    for f in flist:
        if f in blacklist:
            continue
        os.system(f"cp -r ../{f} .")
    def check_image_exists(image_name):
        result = subprocess.run(['docker', 'images', '-q', image_name], stdout=subprocess.PIPE)
        return len(result.stdout.decode('utf-8').strip()) > 0
    if not check_image_exists("telego_build"):
        # os_system_sure("docker pull ubuntu:18.04")
        os_system_sure("docker build -t telego_build .")
    recover_proxy()
    os.chdir("..")
    os.system("rm -rf build")
    
    # run: mount ../telego to /telego, workdir is /telego
    telego_prj_abs=os.path.abspath(".")
    print("building telego at :",telego_prj_abs)
    # os_system_sure(f"docker run -v {telego_prj_abs}:/telego -w /telego telego_build bash -c 'python3 2.build.py '")
    randname="telego_build_"+str(random.randint(10000000,99999999))
    os_system_sure(f"docker run --name {randname} -it  -v {telego_prj_abs}:/telego -w /telego telego_build bash -c 'python3 2.build.py -- privilege'")
    # remove container
    os_system_sure(f"docker rm -f {randname}")
    # end of call the docker
    exit(0)


print("正在容器 ubuntu 18.04 中编译")

# 定义源文件和输出目录
OUTPUT_DIR = "./dist"
CHECKSUM_FILE = "./dist/checksums.txt"

# 创建输出目录
if not os.path.exists(OUTPUT_DIR):
    os.makedirs(OUTPUT_DIR)

# 交叉编译目标平台
targets = [
    ("windows", "amd64"),
    # ("windows", "386"),
    ("linux", "amd64"),
    ("linux", "arm64"),
    # ("darwin", "arm64"),
    # ("darwin", "arm64")
]

# 用于存储校验码的字典
checksums = {}

# 遍历目标平台并进行编译
for goos, goarch in targets:
    output_file = os.path.join(OUTPUT_DIR, f"telego_{goos}_{goarch}")
    if goos == "windows":
        output_file += ".exe"
    
    print(f"Building for {goos}/{goarch}...")
    
    # 构建命令
    cmd = [
        "go", "build",
        "-o", output_file,
        "-buildvcs=false",
        "."
    ]
    
    # 设置环境变量
    env = os.environ.copy()
    env["GOOS"] = goos
    env["GOARCH"] = goarch
    env["GOMODCACHE"] = "/telego/.cache/gomodcache"
    env["GOCACHE"] = "/telego/.cache/gocache"
    
    # 执行命令
    result = subprocess.run(cmd, env=env)
    
    if result.returncode == 0:
        print(f"Successfully built {output_file}")
        
        # 计算校验码
        with open(output_file, "rb") as f:
            sha256_hash = hashlib.sha256(f.read()).hexdigest()
        
        checksums[output_file] = sha256_hash
        print(f"SHA-256 checksum for {output_file}: {sha256_hash}")
    else:
        print(f"Failed to build {output_file}")
        print(f"Error: {result.stderr}")

# 将校验码保存到文件
with open(CHECKSUM_FILE, "w") as f:
    for output_file, checksum in checksums.items():
        output_file=os.path.basename(output_file)
        f.write(f"{output_file}  {checksum}\n")

print(f"Checksums saved to {CHECKSUM_FILE}")



with open("dist/install.py", 'w') as f:
    f.write(f'FILESERVER="http://{main_node_ip}:8003/bin_telego"'+"""
            
import os, tempfile, urllib.request
# if is windows, download to tempdir, move to C:\System32\telego
if os.name == 'nt':
    tempdir=os.path.expanduser("~")+"\\\\telego_install"
    try:
        os.makedirs(tempdir)
    except:
        pass
    os.system(f"curl -L {FILESERVER}/telego_windows_amd64.exe -o {tempdir}\\\\telego.exe")
    os.system(f"move {tempdir}\\\\telego.exe C:\\\\Windows\\\\System32\\\\telego.exe")
else:
    import pwd
    tempdir=tempfile.mkdtemp()
    # check root
    prefix=""
    if os.geteuid() != 0:
        prefix="sudo "
            
    ARCH=""
    if os.uname().machine == "aarch64":
        ARCH="arm64"
    if os.uname().machine == "x86_64":
        ARCH="amd64"
    if os.uname().machine == "arm64":
        ARCH="arm64"
    os.chdir(tempdir)
    urllib.request.urlretrieve(f"{FILESERVER}/telego_linux_{ARCH}", "telego")
    os.system("chmod +x telego")
    curuser=pwd.getpwuid(os.getuid()).pw_name
    os.system(f"{prefix}mv telego /usr/bin/telego")
    os.system(f"{prefix}chown {curuser} /usr/bin/telego")
    
""")
    
with open("dist/install.sh", "w") as f:
    f.write(f"""
#!/bin/bash
FILESERVER="http://{main_node_ip}:8003/bin_telego"
"""+"""
# Check OS type
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    ARCH=""
    MACHINE_TYPE=$(uname -m)

    if [ "$MACHINE_TYPE" == "aarch64" ]; then
        ARCH="arm64"
    elif [ "$MACHINE_TYPE" == "x86_64" ]; then
        ARCH="amd64"
    fi

    TEMP_DIR=$(mktemp -d)

    # Check if root
    if [ $(id -u) -ne 0 ]; then
        PREFIX="sudo"
    else
        PREFIX=""
    fi

    # Download and install telego
    cd $TEMP_DIR
    curl -L "$FILESERVER/telego_linux_${ARCH}" -o telego
    chmod +x telego

    CUR_USER=$(whoami)
    $PREFIX mv telego /usr/bin/telego
    $PREFIX chown $CUR_USER /usr/bin/telego
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    TEMP_DIR="$HOME\\telego_install"

    # Create temporary directory
    mkdir -p $TEMP_DIR

    # Download telego for Windows
    curl -L "$FILESERVER/telego_windows_amd64.exe" -o "$TEMP_DIR\\telego.exe"

    # Move to System32
    mv "$TEMP_DIR\\telego.exe" "C:\\Windows\\System32\\telego.exe"
else
    echo "Unsupported OS"
    exit 1
fi

""")
    
with open("dist/install.sh", "w") as f:
    f.write(f"""
#!/bin/bash
FILESERVER="http://{main_node_ip}:8003/bin_telego"
"""+"""
# Check OS type
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    ARCH=""
    MACHINE_TYPE=$(uname -m)

    if [ "$MACHINE_TYPE" == "aarch64" ]; then
        ARCH="arm64"
    elif [ "$MACHINE_TYPE" == "x86_64" ]; then
        ARCH="amd64"
    fi

    TEMP_DIR=$(mktemp -d)

    # Check if root
    if [ $(id -u) -ne 0 ]; then
        PREFIX="sudo"
    else
        PREFIX=""
    fi

    # Download and install telego
    cd $TEMP_DIR
    curl -L "$FILESERVER/telego_linux_${ARCH}" -o telego
    chmod +x telego

    CUR_USER=$(whoami)
    $PREFIX mv telego /usr/bin/telego
    $PREFIX chown $CUR_USER /usr/bin/telego
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    TEMP_DIR="$HOME\\telego_install"

    # Create temporary directory
    mkdir -p $TEMP_DIR

    # Download telego for Windows
    curl -L "$FILESERVER/telego_windows_amd64.exe" -o "$TEMP_DIR\\telego.exe"

    # Move to System32
    mv "$TEMP_DIR\\telego.exe" "C:\\Windows\\System32\\telego.exe"
else
    echo "Unsupported OS"
    exit 1
fi

""")