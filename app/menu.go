package app
var MenuTreeData = `children:
- comment: "\u76EE\u5F55 - \u5B89\u88C5\u3001\u90E8\u7F72\u6A21\u677F"
  name: deploy-templete
- children:
  - comment: "\u5206\u5E03\u5F0F\u7CFB\u7EDF - \u8F7B\u91CF\u5316k8s"
    name: dist_k3s
  comment: "\u76EE\u5F55 - \u9879\u76EE\u8DEF\u5F84\u4E0B\u7684\u81EA\u5B9A\u4E49\u90E8\
    \u7F72\u9879\u76EE"
  name: deploy
- children:
  - comment: "\u83B7\u53D6\u7BA1\u7406\u5458all in one kubeconfig"
    name: fetch_admin_kubeconfig
  - comment: "\u811A\u672C - \u4E00\u952E\u8BBE\u7F6E\u7535\u4FE1\u5185\u90E8host"
    name: add_tele_host.py
  - comment: "\u811A\u672C - \u4E00\u952E\u8BBE\u7F6Epip\u6E90"
    name: set_pip_source.py
  - comment: "\u811A\u672C - \u4E00\u952E\u8BBE\u7F6Eyum\u6E90"
    name: set_rpm_source.py
  - comment: "\u811A\u672C - \u4E00\u952E\u8BBE\u7F6Eapt\u6E90"
    name: set_apt_source.py
  - comment: "\u542F\u52A8\u4E3B\u8282\u70B9\u6587\u4EF6\u670D\u52A1\u5668"
    name: start_mainnode_fileserver
  - children:
    - comment: "\u751F\u6210/\u83B7\u53D6\u516C\u79C1\u94A5\uFF0Cssh\u514D\u5BC6"
      name: 1.gen_or_get_key
    - comment: "\u5C06\u516C\u94A5\u4E0A\u4F20\u5230\u96C6\u7FA4"
      name: 2.update_key_to_cluster
    comment: "\u914D\u7F6E\u96C6\u7FA4ssh\u514D\u5BC6"
    name: ssh_config
  comment: "\u76EE\u5F55 - \u66F4\u65B0\u914D\u7F6E"
  name: update_config
- comment: "\u5C06\u672C\u5730\u6700\u65B0\u6A21\u677F\u4E0A\u4F20\u5230\u7F51\u7EDC\
    \u5185\u90E8"
  name: deploy-templete-upload
name: "\u4E3B\u83DC\u5355"
`