package generated
type FilterFieldSchema struct {
    Field FieldSchema`json:"field" form:"field" desc:"字段定义"` // 字段定义
    SupportedMatchModes []*MatchMode`json:"supported_match_modes" form:"supported_match_modes" desc:"字段可用的匹配模式"` // 字段可用的匹配模式
}

// 服务树节点信息
type ServiceTreeNode struct {
    ID string`json:"id" form:"id" desc:""`
    Name string`json:"name" form:"name" desc:"节点名称"` // 节点名称
}

type AuthInfo struct {
    Name string`json:"name" form:"name" desc:"签算方式名称"` // 签算方式名称
    Description string`json:"description" form:"description" desc:"具体的签算的描述, 包含算法"` // 具体的签算的描述, 包含算法
    Params []AuthParam`json:"params" form:"params" desc:"参数描述"` // 参数描述
}

// 域名缓存规则
type DomainCacheRule struct {
    Rule string`json:"rule" form:"rule" desc:"规则"` // 规则
    Timeout int32`json:"timeout" form:"timeout" desc:"缓存时间, 单位（秒）"` // 缓存时间, 单位（秒）
    Kind string`json:"kind" form:"kind" desc:"缓存类型: default, suffix, directory, full_path"` // 缓存类型: default, suffix, directory, full_path
}

type DomainPerPlatformOriginServerAuth struct {
    Platform *Platform`json:"platform" form:"platform" desc:"目标平台, 若不是分平台配置，则此字段不存在"` // 目标平台, 若不是分平台配置，则此字段不存在
    AuthInfo AuthInfo`json:"auth_info" form:"auth_info" desc:"鉴权信息，若不开启鉴权，则需传递一个特殊的NoAuth"` // 鉴权信息，若不开启鉴权，则需传递一个特殊的NoAuth
}

// 域名https配置信息
type DomainHttpsConfig struct {
    Enabled bool`json:"enabled" form:"enabled" desc:"是否开启https"` // 是否开启https
    Forced bool`json:"forced" form:"forced" desc:"是否强制https"` // 是否强制https
    EnablePerPlatform bool`json:"enable_per_platform" form:"enable_per_platform" desc:"是否开启分平台配置证书"` // 是否开启分平台配置证书
    Certificates []DomainPlatformCertificate`json:"certificates" form:"certificates" desc:"证书配置，若未开启分平台配置，则只能有一个证书"` // 证书配置，若未开启分平台配置，则只能有一个证书
}

// 域名标签
type AccelerDomainTag struct {
    Name string`json:"name" form:"name" desc:"标签名称"` // 标签名称
    Desc string`json:"desc" form:"desc" desc:"标签描述"` // 标签描述
}

// 子域名配置
type SubAccelerDomain struct {
    ID string`json:"id" form:"id" desc:""`
    Name string`json:"name" form:"name" desc:"域名名称"` // 域名名称
    Core DomainCoreConfig`json:"core" form:"core" desc:"业务信息"` // 业务信息
    Distribute DomainDistributeConfig`json:"distribute" form:"distribute" desc:"分发平台"` // 分发平台
    Cdn DomainCdnConfig`json:"cdn" form:"cdn" desc:"域名cdn配置"` // 域名cdn配置
    Scheduler SubDomainSchedulerConfig`json:"scheduler" form:"scheduler" desc:"域名调度器配置"` // 域名调度器配置
}

type FieldSchema struct {
    Field string`json:"field" form:"field" desc:"字段名称"` // 字段名称
    Title string`json:"title" form:"title" desc:"字段显示名称"` // 字段显示名称
    ValueType string`json:"value_type" form:"value_type" desc:"字段类型: string"` // 字段类型: string
    Options []DisplayedOption`json:"options" form:"options" desc:"字段选项"` // 字段选项
}

type DomainCdnConfig struct {
    OriginServer DomainOriginServerConfig`json:"origin_server" form:"origin_server" desc:"源站信息"` // 源站信息
    BackToOrigin DomainBackToOriginConfig`json:"back_to_origin" form:"back_to_origin" desc:"回源配置"` // 回源配置
    AccessControl DomainAccessControlConfig`json:"access_control" form:"access_control" desc:"访问控制"` // 访问控制
    Https DomainHttpsConfig`json:"https" form:"https" desc:"https配置"` // https配置
    Cache DomainCacheConfig`json:"cache" form:"cache" desc:"缓存配置"` // 缓存配置
    Other DomainOtherConfig`json:"other" form:"other" desc:"用户自定义配置"` // 用户自定义配置
}

type KeyValuePair struct {
    Key string`json:"key" form:"key" desc:""`
    Value string`json:"value" form:"value" desc:""`
}

// 域名在某个平台上的共享缓存配置
type DomainPerPlatformShareCache struct {
    Platform *Platform`json:"platform" form:"platform" desc:"平台"` // 平台
    Domain string`json:"domain" form:"domain" desc:"共享缓存的域名"` // 共享缓存的域名
}

// 基础线路
type BaseLine struct {
    ID string`json:"id" form:"id" desc:"id"` // id
    Name string`json:"name" form:"name" desc:"名称"` // 名称
}

// 域名业务信息
type DomainCoreConfig struct {
    ServiceTreeNode ServiceTreeNode`json:"service_tree_node" form:"service_tree_node" desc:"服务树节点"` // 服务树节点
    BusinessType string`json:"business_type" form:"business_type" desc:"域名的业务类型: image, play, download, dynamic, live, upload, other"` // 域名的业务类型: image, play, download, dynamic, live, upload, other
    BillType string`json:"bill_type" form:"bill_type" desc:"域名的账单类型: vod, imagex, fusion_cdn"` // 域名的账单类型: vod, imagex, fusion_cdn
    Regions []string`json:"regions" form:"regions" desc:"服务区域: china, other"` // 服务区域: china, other
    IsMain bool`json:"is_main" form:"is_main" desc:"是否是主域名"` // 是否是主域名
    IsSingleForm *bool`json:"is_single_form" form:"is_single_form" desc:"域名形态, true: 单域名, false: 多域名。子域名无此字段"` // 域名形态, true: 单域名, false: 多域名。子域名无此字段
    Tags []string`json:"tags" form:"tags" desc:"标签, 只允许最多一个标签"` // 标签, 只允许最多一个标签
    TestUri string`json:"test_uri" form:"test_uri" desc:"测试资源"` // 测试资源
    Owner string`json:"owner" form:"owner" desc:"域名负责人"` // 域名负责人
    CreatedAt int32`json:"created_at" form:"created_at" desc:"创建时间"` // 创建时间
    UpdatedAt int32`json:"updated_at" form:"updated_at" desc:"更新时间"` // 更新时间
    Version int32`json:"version" form:"version" desc:"数据版本号"` // 数据版本号
    Status string`json:"status" form:"status" desc:"域名状态"` // 域名状态
}

// 域名分发平台
type DomainDistributeConfig struct {
    Items []DomainDistributeEntry`json:"items" form:"items" desc:""`
}

// 域名根据状态码进行缓存的规则
type DomainStatusCacheRule struct {
    Code int32`json:"code" form:"code" desc:"规则"` // 规则
    Timeout int32`json:"timeout" form:"timeout" desc:"缓存时间, 单位（秒）"` // 缓存时间, 单位（秒）
}

// 域名抽象接口，可以为主域名或者子域名
type AccelerDomain struct {
    ID string`json:"id" form:"id" desc:"id"` // id
    Name string`json:"name" form:"name" desc:"域名名称"` // 域名名称
    Core DomainCoreConfig`json:"core" form:"core" desc:"业务信息"` // 业务信息
    Distribute DomainDistributeConfig`json:"distribute" form:"distribute" desc:"分发平台"` // 分发平台
    Cdn DomainCdnConfig`json:"cdn" form:"cdn" desc:"cdn配置信息"` // cdn配置信息
}

type Response struct {
    Code int32`json:"code" form:"code" desc:""`
    Message string`json:"message" form:"message" desc:""`
    TraceID *string`json:"trace_id" form:"trace_id" desc:""`
}

// 域名在某个平台上的证书
type DomainPlatformCertificate struct {
    Platform *Platform`json:"platform" form:"platform" desc:"平台"` // 平台
    Certificate Certificate`json:"certificate" form:"certificate" desc:"证书"` // 证书
}

// 基础源站
type BaseOrigin struct {
    ID string`json:"id" form:"id" desc:"id"` // id
    Name string`json:"name" form:"name" desc:"名称"` // 名称
}

// 主域名配置
type MainAccelerDomain struct {
    ID string`json:"id" form:"id" desc:""`
    Name string`json:"name" form:"name" desc:"域名名称"` // 域名名称
    Core DomainCoreConfig`json:"core" form:"core" desc:"业务信息"` // 业务信息
    Distribute DomainDistributeConfig`json:"distribute" form:"distribute" desc:"分发平台信息"` // 分发平台信息
    Cdn DomainCdnConfig`json:"cdn" form:"cdn" desc:"cdn配置信息"` // cdn配置信息
    Scheduler MainDomainSchedulerConfig`json:"scheduler" form:"scheduler" desc:""`
}

type DomainBackToOriginConfig struct {
    Host string`json:"host" form:"host" desc:"回源host"` // 回源host
    Scheme string`json:"scheme" form:"scheme" desc:"回源协议: http, https, follow(保持原协议)"` // 回源协议: http, https, follow(保持原协议)
    Rewrites []RewriteRule`json:"rewrites" form:"rewrites" desc:"回源改写"` // 回源改写
    Headers []KeyValuePair`json:"headers" form:"headers" desc:"回源请求http头部"` // 回源请求http头部
    EnableRange bool`json:"enable_range" form:"enable_range" desc:"是否开启range回源"` // 是否开启range回源
    EnableFollow302 bool`json:"enable_follow_302" form:"enable_follow_302" desc:"是否follow 302"` // 是否follow 302
}

type RewriteRule struct {
    Pattern string`json:"pattern" form:"pattern" desc:"模式"` // 模式
    Replace string`json:"replace" form:"replace" desc:"改写"` // 改写
}

// 域名缓存配置
type DomainCacheConfig struct {
    Rules []DomainCacheRule`json:"rules" form:"rules" desc:"cdn缓存规则"` // cdn缓存规则
    Status []DomainStatusCacheRule`json:"status" form:"status" desc:"根据状态码缓存"` // 根据状态码缓存
    RemoveQueryParams bool`json:"remove_query_params" form:"remove_query_params" desc:"是否去问号缓存"` // 是否去问号缓存
    PersistParams []string`json:"persist_params" form:"persist_params" desc:"保留特定参数"` // 保留特定参数
    EnablePerPlatform bool`json:"enable_per_platform" form:"enable_per_platform" desc:"开启风平台配置共享缓存"` // 开启风平台配置共享缓存
    Shares []DomainPerPlatformShareCache`json:"shares" form:"shares" desc:"共享缓存配置"` // 共享缓存配置
    Headers []KeyValuePair`json:"headers" form:"headers" desc:"http头部(response header)"` // http头部(response header)
    Rewrites []RewriteRule`json:"rewrites" form:"rewrites" desc:"url改写"` // url改写
}

// 过滤域名响应
type FilterAccelerDomainResp struct {
    Domains []AccelerDomain`json:"domains" form:"domains" desc:"域名列表"` // 域名列表
    Pagination PagingResult`json:"pagination" form:"pagination" desc:"分页结果"` // 分页结果
}

// 匹配模式
type MatchMode struct {
    Mode string`json:"mode" form:"mode" desc:"匹配类型: contain, not_contain, equal, not_equal"` // 匹配类型: contain, not_contain, equal, not_equal
    DisplayName string`json:"display_name" form:"display_name" desc:"匹配类型对应的展示名称"` // 匹配类型对应的展示名称
}

type DisplayedOption struct {
    Value string`json:"value" form:"value" desc:"选项值"` // 选项值
    DisplayName string`json:"display_name" form:"display_name" desc:"选项值展示给用户的名称"` // 选项值展示给用户的名称
}

type AuthParam struct {
    Name string`json:"name" form:"name" desc:"参数名"` // 参数名
    Value string`json:"value" form:"value" desc:"参数值"` // 参数值
    Description string`json:"description" form:"description" desc:"参数描述"` // 参数描述
}

// 域名cdn防盗链配置
type DomainReferAuth struct {
    Enabled bool`json:"enabled" form:"enabled" desc:"是否开启防盗链"` // 是否开启防盗链
    IsWhiteMode bool`json:"is_white_mode" form:"is_white_mode" desc:"当前是否是白名单模式"` // 当前是否是白名单模式
    Values []string`json:"values" form:"values" desc:"IP列表"` // IP列表
    AllowEmpty bool`json:"allow_empty" form:"allow_empty" desc:"是否允许空白refer"` // 是否允许空白refer
}

// 域名自定义配置
type DomainOtherConfig struct {
    Items []string`json:"items" form:"items" desc:""`
}

// 过滤域名标签响应结果
type FilterAccelerDomainTagResp struct {
    Tags []AccelerDomainTag`json:"tags" form:"tags" desc:"域名列表"` // 域名列表
    Pagination PagingResult`json:"pagination" form:"pagination" desc:"分页结果"` // 分页结果
}

type DomainOriginServerRule struct {
    MatchRule string`json:"match_rule" form:"match_rule" desc:"匹配规则"` // 匹配规则
    Main DomainOriginServerInfo`json:"main" form:"main" desc:"主源站"` // 主源站
    Backup *DomainOriginServerInfo`json:"backup" form:"backup" desc:"备源站"` // 备源站
}

type DomainAccessControlConfig struct {
    Refer DomainReferAuth`json:"refer" form:"refer" desc:"防盗链"` // 防盗链
    Url DomainUrlAuth`json:"url" form:"url" desc:"url鉴权"` // url鉴权
    IP DomainIPAuth`json:"ip" form:"ip" desc:"ip黑白名单"` // ip黑白名单
}

type Origin struct {
    ID string`json:"id" form:"id" desc:"id"` // id
    Name string`json:"name" form:"name" desc:"名称"` // 名称
}

type DomainOriginServerConfig struct {
    Items []DomainOriginServerRule`json:"items" form:"items" desc:"源站规则列表"` // 源站规则列表
}

// 域名cdn IP黑白名单配置
type DomainIPAuth struct {
    Enabled bool`json:"enabled" form:"enabled" desc:"是否开启黑白名单"` // 是否开启黑白名单
    IsWhiteMode bool`json:"is_white_mode" form:"is_white_mode" desc:"当前是否是白名单模式"` // 当前是否是白名单模式
    Values []string`json:"values" form:"values" desc:"IP列表"` // IP列表
}

// The Query type represents all of the entry points into the API.
type Query struct {
    MGetMainAccelerDomain []MainAccelerDomain`json:"MGetMainAccelerDomain" form:"MGetMainAccelerDomain" desc:"根据id批量获取一组加速主域名的详情"` // 根据id批量获取一组加速主域名的详情
    MGetSubAccelerDomain []SubAccelerDomain`json:"MGetSubAccelerDomain" form:"MGetSubAccelerDomain" desc:"根据id批量获取一组加速子域名的详情"` // 根据id批量获取一组加速子域名的详情
    FilterAccelerDomain FilterAccelerDomainResp`json:"FilterAccelerDomain" form:"FilterAccelerDomain" desc:"过滤域名列表"` // 过滤域名列表
    ListAccelerDomainFilterableField []FilterFieldSchema`json:"ListAccelerDomainFilterableField" form:"ListAccelerDomainFilterableField" desc:"获取域名支持过滤的字段信息"` // 获取域名支持过滤的字段信息
    MGetPlatform []Platform`json:"MGetPlatform" form:"MGetPlatform" desc:"根据id批量获取一组平台的详情"` // 根据id批量获取一组平台的详情
    FilterAccelerDomainTag FilterAccelerDomainTagResp`json:"FilterAccelerDomainTag" form:"FilterAccelerDomainTag" desc:"根据名称过滤标签, 若keyword为null或空字符串, 则返回所有标签"` // 根据名称过滤标签, 若keyword为null或空字符串, 则返回所有标签
}

// 融合线路
type FusionLine struct {
    ID string`json:"id" form:"id" desc:"id"` // id
    Name string`json:"name" form:"name" desc:"名称"` // 名称
}

type DomainDistributeEntry struct {
    Domain *SubAccelerDomain`json:"domain" form:"domain" desc:"域名, 在主域名多域名调度方式下存在，在其它情况下不存在"` // 域名, 在主域名多域名调度方式下存在，在其它情况下不存在
    Platform Platform`json:"platform" form:"platform" desc:"平台"` // 平台
    BaseLine BaseLine`json:"base_line" form:"base_line" desc:"基础线路"` // 基础线路
    BaseLineCname string`json:"base_line_cname" form:"base_line_cname" desc:"基础线路CName"` // 基础线路CName
}

// 融合源站
type FusionOrigin struct {
    ID string`json:"id" form:"id" desc:"id"` // id
    Name string`json:"name" form:"name" desc:"名称"` // 名称
}

// 分页结果
type PagingResult struct {
    PageSize int32`json:"page_size" form:"page_size" desc:"每页条目数量"` // 每页条目数量
    PageNum int32`json:"page_num" form:"page_num" desc:"起始页码，从1开始"` // 起始页码，从1开始
    Total int32`json:"total" form:"total" desc:"总页数"` // 总页数
}

type DomainOriginServerInfo struct {
    IsFusionOrigin bool`json:"is_fusion_origin" form:"is_fusion_origin" desc:"true 为融合源站，false 则为基础源站"` // true 为融合源站，false 则为基础源站
    FusionOrigin *FusionOrigin`json:"fusion_origin" form:"fusion_origin" desc:"绑定的融合源站，当is_fusion_origin=true时有效"` // 绑定的融合源站，当is_fusion_origin=true时有效
    BaseOrigin *BaseOrigin`json:"base_origin" form:"base_origin" desc:"绑定的基础源站，当is_fusion_origin=false时有效"` // 绑定的基础源站，当is_fusion_origin=false时有效
    Endpoint string`json:"endpoint" form:"endpoint" desc:"源站地址"` // 源站地址
    EnablePerPlatform bool`json:"enable_per_platform" form:"enable_per_platform" desc:"开启分平台配置回源鉴权信息"` // 开启分平台配置回源鉴权信息
    Auths []DomainPerPlatformOriginServerAuth`json:"auths" form:"auths" desc:"回源鉴权配置"` // 回源鉴权配置
}

// cdn平台
type Platform struct {
    ID string`json:"id" form:"id" desc:""`
    Name string`json:"name" form:"name" desc:"平台名称"` // 平台名称
}

// 域名cdn防盗链配置
type DomainUrlAuth struct {
    Auth AuthInfo`json:"auth" form:"auth" desc:"鉴权信息"` // 鉴权信息
}

// 主域名调度配置
type MainDomainSchedulerConfig struct {
    ScheduleType string`json:"schedule_type" form:"schedule_type" desc:"调度方式: dns, multi_domain"` // 调度方式: dns, multi_domain
    FusionLineBindings []FusionLineSubDomainBindingEntry`json:"fusion_line_bindings" form:"fusion_line_bindings" desc:"融合线路对应基础线路/子域名绑定关系"` // 融合线路对应基础线路/子域名绑定关系
    Region string`json:"region" form:"region" desc:"区域: china, other"` // 区域: china, other
}

type BaseLineSubDomainBindingEntry struct {
    BaseLine BaseLine`json:"base_line" form:"base_line" desc:"基础线路"` // 基础线路
    SubDomains []SubAccelerDomain`json:"sub_domains" form:"sub_domains" desc:"基础线路绑定的子域名列表"` // 基础线路绑定的子域名列表
}

type FusionLineSubDomainBindingEntry struct {
    FusionLine FusionLine`json:"fusion_line" form:"fusion_line" desc:"融合线路"` // 融合线路
    BaseLineBindings []BaseLineSubDomainBindingEntry`json:"base_line_bindings" form:"base_line_bindings" desc:"融合线路下个基础线路与子域名绑定关系"` // 融合线路下个基础线路与子域名绑定关系
}

type Mutation struct {
    CreateMainAccelerDomain Response`json:"CreateMainAccelerDomain" form:"CreateMainAccelerDomain" desc:"创建主域名"` // 创建主域名
    CreateSubAccelerDomain Response`json:"CreateSubAccelerDomain" form:"CreateSubAccelerDomain" desc:"创建子域名"` // 创建子域名
    MUpdateMainAccelerDomain Response`json:"MUpdateMainAccelerDomain" form:"MUpdateMainAccelerDomain" desc:"批量全量更新主域名"` // 批量全量更新主域名
    MUpdateSubAccelerDomain Response`json:"MUpdateSubAccelerDomain" form:"MUpdateSubAccelerDomain" desc:"批量全量更新子域名"` // 批量全量更新子域名
    MDeleteAccelerDomain Response`json:"MDeleteAccelerDomain" form:"MDeleteAccelerDomain" desc:"批量删除域名"` // 批量删除域名
    CreateAccelerDomainTag Response`json:"CreateAccelerDomainTag" form:"CreateAccelerDomainTag" desc:"创建域名标签。name:标签名称, desc:备注"` // 创建域名标签。name:标签名称, desc:备注
}

// 证书
type Certificate struct {
    ID string`json:"id" form:"id" desc:"id"` // id
    Name string`json:"name" form:"name" desc:"名称"` // 名称
}

// 子域名调度配置
type SubDomainSchedulerConfig struct {
    P2pSupplierID *int32`json:"p2p_supplier_id" form:"p2p_supplier_id" desc:"p2p供应商id"` // p2p供应商id
}
