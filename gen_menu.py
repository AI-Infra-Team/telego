import yaml,os,base64

curfdir=os.path.dirname(os.path.abspath(__file__))
os.chdir(curfdir)
with open("compile_conf.yml", encoding='utf-8') as f:
    conf=yaml.safe_load(f)
main_node_ip=conf["main_node_ip"]
main_node_user=conf["main_node_user"]
main_node_sshport=conf.get("main_node_sshport", "22")  # Default to 22 if not specified
img_repo=conf["image_repo_with_prefix"]

with open("util/compile_conf_gen.go", "w", encoding='utf-8') as f:
    f.write(f"""
package util
var MainNodeIp = "{main_node_ip}"
var MainNodeUser = "{main_node_user}"
var MainNodeSshPort = "{main_node_sshport}"
var ImgRepoAddressWithPrefix = "{img_repo}"
""")

with open("compile_conf.tmp.yml", encoding='utf-8') as f:
    menu_tree_data=yaml.safe_load(f)['menu']

# def walk_menu_tree_data(node,depth):
#     allow_list=[
#         "override comment"
#     ]
#     if depth==2:
#         allow=False
#         for allow_match in allow_list:
#             if allow_match in node:
#                 allow=True
#                 break
#         if not allow:
#             node["forbidden old"]=None
#     for n in node["children"]:
#         walk_menu_tree_data(n,depth+1)
# walk_menu_tree_data(menu_tree_data,0)

def node_add_child(node,child):
    # if"forbidden old" in node:
    #     return False

        # for (idx,child_) in enumerate(node["children"]):
        #     if child_["name"]==child["name"]:
        #         node['children'][idx]=child
        #         child["comment"]=child["comment"]
        # return True
    if 'children' not in node:
        node["children"]=[]
    for idx,child_ in enumerate(node["children"]):
        if child_["name"]==child["name"]:
            old=child_
            node["children"][idx]=child
            if 'comment' in old:
                child["comment"]=old["comment"]
            if 'children' in old:
                child["children"]=old["children"]
            return True            
            # return False
    node["children"].append(child)
    return True

# fill up the tree according to parent dir

# ../teleyard-templete
template_dir=os.path.abspath(os.path.join(curfdir,"teleyard-template"))
print("start walk dir:",template_dir)
for root,dirs,files in os.walk(template_dir):
    relroot=root.replace(os.path.abspath(template_dir)+"/","")
    # print(relroot)
    if relroot=="deploy-templete":
        allow_begin=[
            "bin_",
            "systemd_",
            "k8s_",
        ]
        dirs_sort=sorted(dirs)
        for dir in dirs_sort:
            print("deploy-templetes",relroot,dir)
            if "deployment.yml" in os.listdir(os.path.join(root,dir)):
                children=[{
                    "name": "generate",
                    "comment":"根据模板生成项目",
                    "children":[]
                }]
                if dir.startswith("bin_"):
                    children.append({
                        "name": "install",
                        "comment":"直接安装该项目",
                        "children":[]
                    })
                node_add_child(
                    menu_tree_data["children"][0],{
                        "name": dir,
                        "comment":"模板项目",
                        "children":children
                    })
        continue
    
    allow_eq=[
        "install",
        "update_config",
    ]
    allow_begin=[
        # ["k8s_",2],
        # ["install/bin_",2],
        # ["install/systemd_",2],
        ["update_config/",100]
    ]
    def allow_dir(path):
        if path in allow_eq:
            return True
        match=False
        for [allow_second,depth] in allow_begin:
            if path.startswith(allow_second):
                match=True
                break
            if len(path.split("/"))>depth:
                return False
        return match
    if not allow_dir(relroot):
        # print("skip dir:",relroot)
        continue

    # find cur node in memu tree
    root_splits=relroot.split("/")
    
    cur_node=menu_tree_data
    
    for root_split in root_splits:
        found=False
        if 'children' in cur_node:
            for child in cur_node["children"]:
                if child["name"]==root_split:
                    cur_node=child
                    found=True
                    break
        if not found:
            print("error: cannot find dir in tree"+root_split)
            exit(1)
            
    dirs=sorted(dirs)
    for d in dirs:
        if not allow_dir(relroot+"/"+d):
            # print("skip dir:",relroot+"/"+d)
            continue
        node_add_child(cur_node,{"name":d, "comment":"目录", "children":[]})
        # cur_node["children"].append({"name":d, "comment":"", "children":[]})
    files=sorted(files)
    contains_deployment_yml=False
    for f in files:
        if f=="deployment.yml":
            contains_deployment_yml=True
            break
    if contains_deployment_yml:
        node_add_child(cur_node,{"name":"gen_temp", "comment":"生成操作项目模板", "children":[
            # {"name":"gen_temp","comment":"生成配置模板，后续执行将对应文件作为参数"},
            # {"name":"cancel","comment":"取消执行"},
        ]})
    for f in files:
        if f.endswith(".py"):
            if contains_deployment_yml:
                node_add_child(cur_node,{"name":f, "comment":"部署、安装脚本", "children":[
                    # {"name":"gen_temp","comment":"生成配置模板，后续执行将对应文件作为参数"},
                    {"name":"cancel","comment":"取消执行"},
                ]})
                
                # cur_node["children"].append()
            else:
                encode_file=base64.b64encode(open(root+"/"+f, encoding='utf-8').read().encode("utf-8")).decode("utf-8")
                node_add_child(cur_node,{"name":f, "comment":"未限定脚本", "children":[
                    {"name":"run","comment":"执行"},
                    {"name":"cancel","comment":"取消执行"},
                    {"name":"embed_script","comment": encode_file}
                ]})

encoded=yaml.dump(menu_tree_data)
with open("app/menu.go", "w", encoding='utf-8') as f:
    # write as a go global value
    f.write("package app\nvar MenuTreeData = `"+encoded+"`")