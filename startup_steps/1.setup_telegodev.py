import os, pwd

CUR_F_DIR=os.path.dirname(os.path.abspath(__file__))
# telego_src_dir=os.path.join(CUR_F_DIR,"src")
os.chdir(CUR_F_DIR)

# whereis bash
bash_path=os.popen("whereis bash").read().split()[1]

def os_system(cmd):
    print(cmd)
    os.system(cmd)

with open("../1.run.sh") as f:
    content=f.read()
    prjdir=os.path.dirname(CUR_F_DIR)
    content=f"export FROM_GO_RUN=true\n"+content
    content=f"cd {prjdir}\n"+content
    content=f"#!{bash_path}\n"+content
with open("telego","w") as f:
    f.write(content)
os_system("chmod +x telego")
rootprefix=""
if os.getuid() != 0:
    rootprefix="sudo "
os_system(f"{rootprefix}mv telego /usr/bin/")

curuser = pwd.getpwuid(os.getuid()).pw_name
os_system(f"{rootprefix}chown {curuser} /usr/bin/telego")
os_system(f"{rootprefix}chmod +x /usr/bin/telego")

print("Updated telego dev, just type telego in command")