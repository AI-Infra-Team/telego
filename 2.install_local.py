import os
import platform

def os_system(cmd):
    print(f"Running command: {cmd}")
    os.system(cmd)

def os_system_sure(cmd):
    print(f"Running command: {cmd}")
    result = os.system(cmd)
    if result != 0:
        print(f"Command failed: {cmd}")
        exit(1)


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

def get_target_bin():
    # get cursys
    cursys = ""
    if os.name == "nt":
        cursys = "windows"
    else:
        cursys = "linux"

    # get curarch
    curarch = ""
    machine = platform.machine().lower()
    if machine in ["x86_64", "amd64"]:
        curarch = "amd64"
    elif machine in ["aarch64", "arm64"]:
        curarch = "arm64"
    else:
        curarch = "amd64"  # default fallback

    targetbin=""
    if os.path.exists(f"./dist/telego_{cursys}_{curarch}"):
        targetbin = f"./dist/telego_{cursys}_{curarch}"
    else:
        targetbin = f"./dist/telego_{cursys}_{curarch}.exe"
    return targetbin

def is_admin_windows():
    """Check if running as administrator on Windows"""
    try:
        import ctypes
        return ctypes.windll.shell32.IsUserAnAdmin()
    except:
        return False

def install_binary():
    targetbin = get_target_bin()
    
    if os.name == "nt":  # Windows
        # On Windows, install to a directory in PATH or create one
        # install_dir = os.path.expanduser("~\\AppData\\Local\\telego")
        # use ://Windows/System32
        # disk
        disk = os.getenv("SystemDrive")
        print(f"disk: {disk}")
        install_dir = os.path.join(f"{disk}\\Windows\\System32")
        
        # Create directory if it doesn't exist
        if not os.path.exists(install_dir):
            os.makedirs(install_dir)
        
        install_path = os.path.join(install_dir, "telego.exe")
        
        # Copy the binary
        import shutil
        try:
            shutil.copy2(targetbin, install_path)
            print(f"Binary installed to: {install_path}")
            
            # Add to PATH if not already there
            print(f"\nTo use telego from anywhere, add this directory to your PATH:")
            print(f"  {install_dir}")
            print("\nOr run this PowerShell command as Administrator:")
            print(f'  $env:PATH += ";{install_dir}"')
            print(f'  [Environment]::SetEnvironmentVariable("PATH", $env:PATH, [EnvironmentVariableTarget]::Machine)')
            
        except PermissionError:
            raise PermissionError(f"Permission denied. Please run as Administrator or manually copy:"+
            f"copy \"{targetbin}\" \"{install_path}\"")
            # print(f"Permission denied. Please run as Administrator or manually copy:")
            # print(f"  copy \"{targetbin}\" \"{install_path}\"")
            
    else:  # Linux/Unix
        # copy targetbin to /usr/bin
        sudo = ""
        try:
            if os.geteuid() != 0:
                sudo = "sudo "
        except AttributeError:
            # geteuid not available, assume we need sudo
            sudo = "sudo "
        
        os_system_sure(f"{sudo}cp {targetbin} /usr/bin/telego")

# os_system_sure("python3 1.build.py")
# execute_python_script("1.build.py")

install_binary()

# set NO_UPGRADE env to avoid auto upgrade
print("\nmaybe u r developing and testing")
print("set NO_UPGRADE env to avoid auto upgrade")

if os.name == "nt":  # Windows
    print("set NO_UPGRADE=true")
    print("# Or in PowerShell:")
    print("$env:NO_UPGRADE = 'true'")
else:  # Linux/Unix
    print("export NO_UPGRADE=true")


