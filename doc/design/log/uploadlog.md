# UploadLog
## 概览
用于dns解析日志上传。用户可以将dns解析日志整体打包通过ftp模式上传到自己的服务器，方便自行查阅解析。

## 设计
上传流程：
* 用户点击FTP导出，输入服务器地址、账号名称、密码。点击导出日志。
* controller通过验证账号密码正确以后通过kafka下发命令到dns。
* dns消费kafka命令将query.log整体打包ftp上传到用户输入的服务器地址。
* 上传过程中通过kafka反馈上传状态。
* 用户打开ftp导出面板定期会刷新状态，当完成时，将显示上传完成时间。

## 资源 （UploadLog）
* 顶级资源包括：UserName（ftp用户名），Password（密码），Address（ftp服务器地址），Status（上传状态），
Comment（备注），FileName（上传完成后的文件名），FinishTime（上传完成时间）。
* 其中UserName、Password、Address是必填。Address为用户ftp服务器地址，包括IP+端口。
* Status为上传状态，一共有：connecting,connectFailed,transporting,transportFailed,completed状态。
* 每个用户同一时间内只能又一个上传任务，并且每个服务器地址同一时间内也只能保持一个上传任务。
