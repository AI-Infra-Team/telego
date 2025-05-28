import os
import hashlib
import subprocess
import yaml
import random
import sys
import shutil
import platform
import importlib.util

# 设置UTF-8编码环境
def setup_global_utf8():
    """设置全局UTF-8编码环境"""
    try:
        # 设置环境变量
        os.environ['PYTHONIOENCODING'] = 'utf-8'
        
        # 重新配置标准输入输出
        if hasattr(sys.stdout, 'reconfigure'):
            sys.stdout.reconfigure(encoding='utf-8')
            sys.stderr.reconfigure(encoding='utf-8')
            if hasattr(sys.stdin, 'reconfigure'):
                sys.stdin.reconfigure(encoding='utf-8')
    except Exception as e:
        print(f"设置UTF-8编码时出现警告: {e}")

# 在脚本开始时设置UTF-8编码
setup_global_utf8()

# 预声明全局变量
HOST_PROJECT_DIR = None
PROJECT_ROOT = None

def execute_python_script(script_path):
    """
    更优雅地执行Python脚本的方法
    优先使用importlib动态导入，如果失败则使用exec
    """
    import sys
    
    # 设置UTF-8编码环境
    def setup_utf8_encoding():
        try:
            if hasattr(sys.stdout, 'reconfigure'):
                sys.stdout.reconfigure(encoding='utf-8')
                sys.stderr.reconfigure(encoding='utf-8')
        except:
            pass
    
    try:
        # 方法1: 使用importlib动态导入（推荐）
        if os.path.exists(script_path):
            spec = importlib.util.spec_from_file_location("gen_menu", script_path)
            if spec and spec.loader:
                gen_menu_module = importlib.util.module_from_spec(spec)
                # 保存当前工作目录
                original_cwd = os.getcwd()
                try:
                    # 设置UTF-8编码环境
                    setup_utf8_encoding()
                    
                    spec.loader.exec_module(gen_menu_module)
                    print(f"成功执行 {script_path} (使用importlib)")
                    return True
                finally:
                    # 恢复原始工作目录
                    os.chdir(original_cwd)
    except Exception as e:
        print(f"使用importlib执行失败: {e}")
        
    try:
        # 方法2: 使用exec作为备选方案
        if os.path.exists(script_path):
            print(f"正在执行 {script_path} (使用exec)")
            # 保存当前工作目录
            original_cwd = os.getcwd()
            
            try:
                # 明确指定UTF-8编码读取文件
                with open(script_path, 'r', encoding='utf-8') as f:
                    script_code = f.read()
                
                # 创建一个新的命名空间来执行脚本
                script_globals = {
                    '__file__': os.path.abspath(script_path),
                    '__name__': '__main__'
                }
                
                # 设置UTF-8编码环境
                setup_utf8_encoding()
                
                # 执行脚本
                exec(script_code, script_globals)
                print(f"成功执行 {script_path} (使用exec)")
                return True
            finally:
                # 恢复原始工作目录
                os.chdir(original_cwd)
    except Exception as e:
        print(f"使用exec执行失败: {e}")
        return False
    
    print(f"警告: 未找到脚本文件 {script_path}")
    return False

# 跨平台文件操作函数
def safe_mkdir(path):
    """跨平台创建目录"""
    try:
        os.makedirs(path, exist_ok=True)
        return True
    except Exception as e:
        print(f"创建目录失败: {path}, 错误: {e}")
        return False

def safe_copy(src, dst):
    """跨平台复制文件或目录"""
    try:
        if os.path.isdir(src):
            if os.path.exists(dst):
                shutil.rmtree(dst)
            shutil.copytree(src, dst)
        else:
            shutil.copy2(src, dst)
        return True
    except Exception as e:
        print(f"复制失败: {src} -> {dst}, 错误: {e}")
        return False

def safe_remove(path):
    """跨平台删除文件或目录"""
    try:
        if os.path.exists(path):
            if os.path.isdir(path):
                shutil.rmtree(path)
            else:
                os.remove(path)
        return True
    except Exception as e:
        print(f"删除失败: {path}, 错误: {e}")
        return False

def safe_chdir(path):
    """跨平台切换目录"""
    try:
        os.chdir(path)
        return True
    except Exception as e:
        print(f"切换目录失败: {path}, 错误: {e}")
        return False

def run_with_sudo(command_list):
    """跨平台执行需要管理员权限的命令"""
    if platform.system() == "Windows":
        # Windows下不需要sudo，直接执行
        return run_command(" ".join(command_list))
    else:
        # Unix/Linux系统使用sudo
        return run_command("sudo " + " ".join(command_list))

def find_project_root():
    """向上查找项目根目录（包含 compile_conf.tmp.yml 的目录）"""
    current_dir = os.path.dirname(os.path.abspath(__file__))
    while True:
        if os.path.exists(os.path.join(current_dir, "compile_conf.tmp.yml")):
            return current_dir
        parent_dir = os.path.dirname(current_dir)
        if parent_dir == current_dir:
            raise RuntimeError("无法找到项目根目录（未找到 compile_conf.tmp.yml）")
        current_dir = parent_dir
def is_in_github_codespace():
    return os.environ.get("CODESPACE_NAME") is not None

# 检查是否在容器内
def is_in_container():
    if is_in_github_codespace():
        return False
        
    # 检查 /.dockerenv 文件
    if os.path.exists('/.dockerenv'):
        return True
    
    # 检查 cgroup
    try:
        with open('/proc/1/cgroup', 'r') as f:
            return 'docker' in f.read()
    except:
        return False

# 获取项目根目录
PROJECT_ROOT = find_project_root()
print(f"项目根目录：{PROJECT_ROOT}")

# 处理 build_context.yml
build_context_path = os.path.join(PROJECT_ROOT, "build_context.yml")
if not is_in_container():
    # 在主机上，更新 build_context.yml
    HOST_PROJECT_DIR = PROJECT_ROOT
    with open(build_context_path, "w") as f:
        yaml.dump({
            "HOST_PROJECT_DIR": HOST_PROJECT_DIR
        }, f, default_flow_style=False)
    print(f"已更新 build_context.yml：{build_context_path}")
else:
    # 在容器内，读取 HOST_PROJECT_DIR
    with open(build_context_path, "r") as f:
        build_context = yaml.safe_load(f)
        HOST_PROJECT_DIR = build_context["HOST_PROJECT_DIR"]
        print(f"从 build_context.yml 读取到主机项目目录：{HOST_PROJECT_DIR}")

# 切换到当前脚本所在目录
curfdir = os.path.dirname(os.path.abspath(__file__))
os.chdir(curfdir)

# debug current path and filelist
print(f"当前路径：{curfdir}")
print(f"当前目录下的文件：{os.listdir(curfdir)}")
# for f in os.listdir(curfdir):
#     print(f"- {f}")

try:
    with open("compile_conf.yml",encoding="utf-8") as f:
        conf=yaml.safe_load(f)
except Exception as e:
    raise Exception(f"读取配置文件 compile_conf.yml 时发生错误: {e}\n"+
        "复制根目录下的 compile_conf.tmp.yml 为compile_conf.yml，并参照文档配置对应项")

main_node_ip=conf["main_node_ip"]
main_node_user=conf["main_node_user"]
# 读取代理配置，如果不存在则为None
proxy_config = conf.get("proxy", None)

def cmd_color(string,color):
    color_dict={"red":31,"green":32,"yellow":33,"blue":34,"magenta":35,"cyan":36,"white":37}
    return f"\033[{color_dict[color]}m{string}\033[0m"

def run_command(command, check=True):
    BEFORE_RUN_TITLE=cmd_color("执行命令：","blue")
    RUN_FAIL_TITLE=cmd_color(">","blue")+"\n"+cmd_color("命令执行失败：","red")
    RUN_SUCCESS_TITLE=cmd_color(">","blue")+"\n"+cmd_color("命令执行成功：","green")
    
    print(f"{BEFORE_RUN_TITLE}{command}")
    ret=os.system(command)
    if ret!=0:
        print(f"{RUN_FAIL_TITLE}{command}")
        print(f"错误输出：\n{ret}")
        if check:
            sys.exit(1)
        return False
    else:
        print(f"{RUN_SUCCESS_TITLE}{command}\n")
        return True
    # try:
    #     result = subprocess.run(command, shell=True, check=check, text=True, capture_output=True)
    #     print(result.stdout)
    #     if result.stderr:
    #         print(result.stderr)
    #     if check and result.returncode != 0:
    #         print(f"{RUN_FAIL_TITLE}{command}")
    #         print(f"错误输出：\n{result.stderr}")
    #         sys.exit(1)
    #     print(f"{RUN_SUCCESS_TITLE}{command}\n")
    #     return result.returncode == 0
    # except subprocess.CalledProcessError as e:
    #     print(f"{RUN_FAIL_TITLE}{command}")
    #     print(f"错误输出：\n{e.stderr}")
    #     if check:
    #         sys.exit(1)
    #     return False

def os_system_sure(command):
    return run_command(command, check=True)

def os_system(command):
    return run_command(command, check=False)

def pull_if_not_present(image: str):
    r = subprocess.run(["docker", "image", "inspect", image], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if r.returncode != 0:
        print(f"Pulling {image}...")
        subprocess.run(["docker", "pull", image], check=True)
    else:
        print(f"{image} already exists locally.")

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
    # 直接执行Python脚本，而不是通过命令行启动
    try:
        # 查找gen_menu.py文件
        gen_menu_path = None
        if os.path.exists("gen_menu.py"):
            gen_menu_path = "gen_menu.py"
        elif os.path.exists(os.path.join(PROJECT_ROOT, "gen_menu.py")):
            gen_menu_path = os.path.join(PROJECT_ROOT, "gen_menu.py")
        
        if gen_menu_path:
            execute_python_script(gen_menu_path)
        else:
            print("警告: 未找到 gen_menu.py 文件")
            
    except Exception as e:
        print(f"警告: 执行 gen_menu.py 时发生错误: {e}")

    print("将启动 ubuntu 18.04进行编译")
    # 使用跨平台函数创建build目录
    safe_mkdir("build")

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
            
    pull_if_not_present("ubuntu:18.04")
    # os_system_sure("docker pull ubuntu:18.04")

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
    
    # 使用跨平台函数切换目录
    safe_chdir("build")
    flist=os.listdir("../")
    blacklist=[".cache",".git",".dist","build","telego_prjs","teleyard-template"]
    for f in flist:
        if f in blacklist:
            continue
        # 使用跨平台函数复制文件
        src_path = os.path.join("..", f)
        dst_path = f
        if not safe_copy(src_path, dst_path):
            print(f"警告: 复制 {f} 失败")
        
    def check_image_exists(image_name):
        result = subprocess.run(['docker', 'images', '-q', image_name], stdout=subprocess.PIPE)
        return len(result.stdout.decode('utf-8').strip()) > 0
    if not check_image_exists("telego_build"):
        # os_system_sure("docker pull ubuntu:18.04")
        os_system_sure("docker build -t telego_build .")
    recover_proxy()
    # 使用跨平台函数切换回上级目录
    safe_chdir("..")
    # 使用跨平台函数删除build目录
    safe_remove("build")
        
    # run: mount HOST_PROJECT_DIR to /telego, workdir is /telego
    print("building telego at host project dir:", HOST_PROJECT_DIR)
    # print(f"files in {HOST_PROJECT_DIR}:")
    # for f in os.listdir(HOST_PROJECT_DIR):
    #     print(f"- {f}")

    randname="telego_build_"+str(random.randint(10000000,99999999))
    os_system_sure(f"docker run --name {randname} -v {HOST_PROJECT_DIR}:/telego -w /telego telego_build bash -c \"python3 1.build.py -- privilege\"")
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


# 设置环境变量
env = os.environ.copy()
env["GOMODCACHE"] = "/telego/.cache/gomodcache"
env["GOCACHE"] = "/telego/.cache/gocache"

# 如果配置了代理，设置Go编译环境代理
if proxy_config:
    env["http_proxy"] = proxy_config
    env["https_proxy"] = proxy_config
    # env["GOPROXY"] = "https://goproxy.cn,direct"  # 使用中国区Go代理

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
    env["GOOS"] = goos
    env["GOARCH"] = goarch
    
    # 开始之前，先执行 go mod tidy
    result = subprocess.run(["go", "mod", "tidy"], env=env)
    if result.returncode != 0:
        print(f"Failed to run go mod tidy")
        print(f"Error: output({result.stdout}), err({result.stderr})")
        sys.exit(1)

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
        print(f"Error: result({result})")
        raise Exception(f"Failed to build {output_file}")

# 将校验码保存到文件
with open(CHECKSUM_FILE, "w") as f:
    for output_file, checksum in checksums.items():
        output_file=os.path.basename(output_file)
        f.write(f"{output_file}  {checksum}\n")

print(f"Checksums saved to {CHECKSUM_FILE}")

with open("dist/install.ps1", 'w', encoding='utf-8') as f:
    f.write(f"""$FILESERVER = "http://{main_node_ip}:8003/bin_telego"
$TEMP_DIR = "$env:USERPROFILE\\telego_install"

# 创建临时目录
if (-not (Test-Path $TEMP_DIR)) {{
    New-Item -ItemType Directory -Path $TEMP_DIR | Out-Null
}}

# 下载文件
$downloadUrl = "$FILESERVER/telego_windows_amd64.exe"
$outputPath = "$TEMP_DIR\\telego.exe"

Write-Host "正在下载 telego..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $outputPath

# 检查下载是否成功
if (Test-Path $outputPath) {{
    Write-Host "下载完成，正在安装到 System32..."
    
    # 复制到 System32
    Copy-Item -Path $outputPath -Destination "C:\\Windows\\System32\\telego.exe" -Force
    
    # 清理临时文件
    Remove-Item -Path $TEMP_DIR -Recurse -Force
    
    Write-Host "安装完成！telego 已安装到 C:\\Windows\\System32\\telego.exe"
}} else {{
    Write-Host "下载失败，请检查网络连接或文件服务器状态。"
    exit 1
}}
""")

with open("dist/install.py", 'w', encoding='utf-8') as f:
    f.write(f'FILESERVER="http://{main_node_ip}:8003/bin_telego"'+"""
            
import os, tempfile, urllib.request, shutil, subprocess, platform
# if is windows, download to tempdir, move to C:\System32\telego
if os.name == 'nt':
    tempdir=os.path.expanduser("~")+"\\\\telego_install"
    try:
        os.makedirs(tempdir, exist_ok=True)
    except:
        pass
    
    # 使用urllib下载文件
    telego_url = f"{FILESERVER}/telego_windows_amd64.exe"
    telego_temp_path = os.path.join(tempdir, "telego.exe")
    telego_target_path = "C:\\\\Windows\\\\System32\\\\telego.exe"
    
    try:
        print(f"正在下载 {telego_url}")
        urllib.request.urlretrieve(telego_url, telego_temp_path)
        print("下载完成，正在安装...")
        
        # 使用shutil.move替代move命令
        shutil.move(telego_temp_path, telego_target_path)
        print(f"安装完成！telego 已安装到 {telego_target_path}")
        
        # 清理临时目录
        shutil.rmtree(tempdir, ignore_errors=True)
    except Exception as e:
        print(f"安装失败: {e}")
        exit(1)
else:
    import pwd
    tempdir=tempfile.mkdtemp()
    
    # check root
    is_root = os.geteuid() == 0
    
    ARCH=""
    machine = os.uname().machine
    if machine == "aarch64" or machine == "arm64":
        ARCH="arm64"
    elif machine == "x86_64":
        ARCH="amd64"
    else:
        print(f"不支持的架构: {machine}")
        exit(1)
    
    try:
        os.chdir(tempdir)
        telego_url = f"{FILESERVER}/telego_linux_{ARCH}"
        telego_path = "telego"
        
        print(f"正在下载 {telego_url}")
        urllib.request.urlretrieve(telego_url, telego_path)
        
        # 设置可执行权限
        os.chmod(telego_path, 0o755)
        
        curuser = pwd.getpwuid(os.getuid()).pw_name
        target_path = "/usr/bin/telego"
        
        if is_root:
            # 如果是root用户，直接移动
            shutil.move(telego_path, target_path)
            os.chown(target_path, pwd.getpwnam(curuser).pw_uid, pwd.getpwnam(curuser).pw_gid)
        else:
            # 如果不是root用户，使用sudo
            subprocess.run(["sudo", "mv", telego_path, target_path], check=True)
            subprocess.run(["sudo", "chown", curuser, target_path], check=True)
        
        print(f"安装完成！telego 已安装到 {target_path}")
        
    except Exception as e:
        print(f"安装失败: {e}")
        exit(1)
    finally:
        # 清理临时目录
        try:
            shutil.rmtree(tempdir, ignore_errors=True)
        except:
            pass
    
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

def load_compile_conf():
    try:
        with open('compile_conf.yml', 'r') as f:
            return yaml.safe_load(f)
    except FileNotFoundError:
        return {}

def set_proxy_from_conf():
    if not os.environ.get("http_proxy"):
        conf = load_compile_conf()
        if conf and 'proxy' in conf:
            proxy = conf['proxy']
            os.environ['http_proxy'] = proxy
            os.environ['https_proxy'] = proxy
            return True
    return False
    # 尝试从配置文件设置代理
proxy_set = set_proxy_from_conf()
