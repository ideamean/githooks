package main

type PHPStyleCheck struct {
	Enable    bool
	PHPCS     string
	PHPCSArgs []interface{}
}

type JSStyleCheck struct {
	Enable bool
	PHPCS  string
}

type GOStyleCheck struct {
	Enable           bool
	GolangCiLint     string
	GolangCiLintArgs []interface{}
}

type StyleCheck struct {
	PHP PHPStyleCheck
	JS  JSStyleCheck
	GO  GOStyleCheck
}

type CommitLogHookHttp struct {
	Enable     bool
	ReceiveURL string
	Header     map[string]string
}

// Conf hook配置
type Conf struct {
	// 是否自动清除缓存文件, 调试可以改为false, 缓存文件路径会打印到header信息中
	ClearCache bool
	// 代码豁免code文件存储路径
	// commit message中携带[A]code[/A], 会检测 $CodeExemptionDir/code 文件是否存在，存在则跳过hooks逻辑并删除该文件
	// 每个code只可使用一次
	CodeExemptionDir string
	// 允许提交的邮箱
	AllowEmail []string
	// 分支保护, 不允许直接推送, 必须通过合并请求推送
	ProtectBranch []string
	// 超级账号, 在该账号是合法的邮箱格式时，该账号不做规则检测
	SuperAccount []string
	// 跳过检查的命名空间
	IgnoreNamespace []string
	// 跳过检查的项目
	IgnoreRepos []string
	// commit message 是否需要带jira号, 正则表达式
	RequireJiraIDRexp string
	// 代码检查
	StyleCheck StyleCheck
	// 代码提交日志收集, 支持http接口接收
	CommitLogHook struct {
		Http CommitLogHookHttp
	}
}
