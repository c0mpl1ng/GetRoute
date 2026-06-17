# GetRoute

Java 部署包静态分析工具，使用 Go 语言开发，无需 JDK 或任何 Java 工具链。

直接解析 `.class` 字节码、XML 配置文件、MANIFEST.MF，自动识别框架并提取 Web 路由、Class 信息、框架信息、组件信息，最终输出为一个 Excel 文件。

## 应用场景

- Java 代码审计
- 红队信息收集
- Java 资产测绘
- Web 路由枚举
- Java 框架识别
- 中间件识别
- 漏洞挖掘前期分析

## 快速开始

```bash
# 编译
./build.sh

# 使用
./bin/GetRoute -input app.jar
./bin/GetRoute -input app.war -output ./result -verbose
./bin/GetRoute -input app.zip -threads 20
```

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-input` | 输入文件路径（jar/war/zip） | 必填 |
| `-output` | 输出目录 | 当前目录 |
| `-threads` | 并发数 | CPU 核心数 |
| `-verbose` | 输出详细日志 | false |

## 输出文件

生成一个 `GetRoute.xlsx`，包含 4 个 Sheet：

### Routes

| 字段 | 说明 |
|------|------|
| URL | 路由路径 |
| HTTP_METHOD | HTTP 方法 |
| FRAMEWORK | 所属框架 |
| CLASS_NAME | 类名 |
| CLASS_PATH | Class 文件路径 |
| METHOD_NAME | 方法名 |
| SOURCE_JAR | 来源 jar |

### Classes

| 字段 | 说明 |
|------|------|
| CLASS_NAME | 完整类名 |
| CLASS_PATH | Class 文件路径 |
| SOURCE_JAR | 来源 jar |

### Framework

| 字段 | 说明 |
|------|------|
| FRAMEWORK | 框架名称 |
| VERSION | 版本 |
| CONFIDENCE | 置信度 (0-100) |
| EVIDENCE | 检测证据 |

### Components

| 字段 | 说明 |
|------|------|
| COMPONENT | 组件名称 |
| TYPE | 组件类型 |
| VERSION | 版本 |
| SOURCE | 来源依据 |

## 支持的框架

| 框架 | 识别方式 |
|------|----------|
| Spring MVC | @Controller/@RestController + @RequestMapping 等注解 |
| Spring Boot | BOOT-INF 目录、spring.factories、MANIFEST.MF |
| Struts2 | struts.xml、@Action、ActionSupport |
| WebWork | xwork.xml、ServletActionContext |
| JAX-RS | @Path、@GET、@POST 等注解 |
| Servlet | @WebServlet、web.xml |
| Eway Framework | xwork_*.xml、com.eway.* 类（东华医疗） |

## 支持的组件识别

Spring MVC、Spring Boot、Spring Cloud、Spring Security、MyBatis、MyBatis Plus、Hibernate、JPA、Struts2、Dubbo、gRPC、Shiro、SaToken、Tomcat、Jetty、Undertow、Log4j、Log4j2、Logback、SLF4J、Jackson、Fastjson、Fastjson2、Gson、Druid、HikariCP、C3P0、Thymeleaf、FreeMarker、Velocity、Beetl、HttpClient、OkHttp、JReap、RuoYi、Jeecg、DHCC、Eway Framework、Guava、Ehcache、Jedis、Lettuce、CXF、Axis2、Swagger、Knife4j、Commons IO/Lang/BeanUtils/Collections/FileUpload、JUnit、TestNG 等 65+ 组件。

## 性能

- 支持 10000+ Class、1000+ Jar、500MB+ 部署包
- 扫描时间 5 分钟以内
- 内存占用 2GB 以内

## 技术特点

- 不依赖 JDK、javap、fernflower、CFR、JADX、JD-GUI
- 不反编译源码后正则匹配
- 直接解析 class 字节码中的 RuntimeVisibleAnnotations
- 支持 jar 中的 jar、war 中的 jar、zip 中的 war/zip 无限递归
- 适配器模式，新增框架只添加一个 Extractor 即可

## 跨平台编译

```bash
# 编译全部平台
./build.sh

# 按平台编译
./build.sh linux
./build.sh darwin
./build.sh windows
```

生成文件：

```
bin/
├── GetRoute                    # 本机
├── GetRoute-darwin-amd64
├── GetRoute-darwin-arm64
├── GetRoute-linux-amd64
├── GetRoute-linux-arm64
└── GetRoute-windows-amd64.exe
```

## 项目结构

```
GetRoute/
├── cmd/getroute/main.go         # CLI 入口
├── internal/
│   ├── model/                   # 数据模型
│   ├── archive/                 # ZIP/JAR/WAR 读取、MANIFEST 解析
│   ├── classfile/               # .class 字节码解析、注解解析
│   ├── scanner/                 # 归档扫描、并发调度
│   ├── xmlconfig/               # web.xml、struts.xml、Spring XML 解析
│   ├── extractor/               # 路由提取器（6 种框架）
│   ├── detector/                # 框架识别、组件识别
│   ├── indexer/                 # 去重、排序、URL 标准化
│   └── exporter/                # Excel 导出
├── build.sh                     # 编译脚本
├── Makefile                     # Make 构建
└── go.mod
```

## 依赖

- Go 1.24+
- [excelize](https://github.com/xuri/excelize) — Excel 生成（唯一外部依赖）
- 其余全部使用 Go 标准库

## License

MIT
