# gom
golang的简单易用ORM工具

## 目标
这个ORM工具希望达到
1. 配置接入简单,易维护
2. SQL完全可控
3. 数据库查询结果和内存对象自动映射
4. 学习成本低
的目标

## 使用方式
### 1.建表
```sql
create table if not exists user (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `name` varchar(32) NOT NULL,
    `age` varchar(32) NOT NULL,
    `create_time` timestamp NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `name` (`name`)
)
```

### 2.创建yaml配置
```yaml
abbreviation:
  table: user
  colum: id, name, age, create_time
query:
  - name: queryById
    sql: select <colum> from <table> where id = ${id}
  - name: queryByName
    sql: select <colum> from <table> where name = ${name}
  - name: queryByAgeGT
    sql: select <colum> from <table> where age > ${age}
  - name: queryAll
    sql: select <colum> from <table>
exec:
  - name: insert
    sql: insert into <table> <colum> values (default, ${name}, ${age), Now()})
```

query字段里面都是select查询语句, exec里面是更新和删除语句. query和exec中的name将在golang代码中使用,在单个yaml文件里面name需要是唯一的.

abbreviation定义的是重复使用的片段, 在sql中出现的`<key>`都会自动替换成abbreviation中定义的值,如`select <colum> from <table>`会被替换为 `select id, name, age, create_time from user`.

### 3.在代码中加载yaml
```golang
const (
	DBUser     = "root"
	DBPassword = "11111111"
	DBIP       = "127.0.0.1"
	DBPort     = "3306"
	Database   = "test"
	DBOptions  = "charset=utf8mb4&parseTime=True&loc=Local"
)

type User struct {
	ID          int64
	Name        string
	Age         int
	CreateTime  time.Time
}

func (u *User) Scan(scanable Scanable) error {
	return scanable.Scan(&u.ID, &u.Name, &u.Age, &u.CreateTime)
}

m, err := LoadModel("model_test.yaml", &User{})
if err != nil {
    // error 处理
}

DBURL := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", DBUser, DBPassword, DBIP, DBPort, Database, DBOptions)
db, err := sql.Open("mysql", DBURL)
if err != nil {
    // 创建数据库连接失败
}
```
定义User结构, 加载yaml文件, 创建数据库连接.
**这里要注意,User必须实现Scan方法, Scan方法的参数顺序和select出来的字段顺序必须一致**

### 4.使用yaml中的sql
```golang
1| users, err := m.Query(db, "queryAll")  
 
2| user, err := m.QueryRow(db, "queryById", 1)
3| user, err := m.QueryRow(db, "queryById", User{ID:1})
4| user, err := m.QueryRow(db, "queryById", &User{ID:1})
 |
5| users, err := m.Query(db, "queryByAgeGT", 10)
6| users, err := m.Query(db, "queryByAgeGT", User{Age:10})
7| users, err := m.Query(db, "queryByAgeGT", &User{Age:10})
 
8| AffectRows, LastInsertID, err = m.Exec(db, "insert", "bob", 35)
9| AffectRows, LastInsertID, err = m.Exec(db, "insert", &User{Name:"bob", Age:35})
```
QueryRow查询一条结果
Query查询结果列表
Exec执行插入,更新,删除操作
参数类型支持结构指针(4,7,9行),结构(3,6行)和变长的基础数据类型(2,5,8行).

看个例子
```yaml
name: queryComplicate
sql: select <column> from <table> where Name = ${name} and Age = ${age}"
```
- 如果是指针或结构,参数绑定规则可以参考下例:
    
    ```m.Query(db, "queryComplicate", &User{Name:"bob", Age:35})```
    
    \${name}会在User中找到Name字段,\${age}则会绑定到User中的Age字段.

- 如果调用方式是用变长的基础数据类型:
    
    ```m.Query(db, "queryComplicate", "bob", 35)```
    
    \${name}将绑定第一个参数"bob", \${age}绑定第二个参数35, 此时函数参数顺序非常重要

### 5.批量插入
go的sql dirver不支持批量插入,gom对此做了增强,一个批量插入例子如下:
```go
func getMockUserList() []*User {
	return []*User{
        &User{Name: "admin", Age: 10},
        &User{Name: "Aabbye", Age: 10},
        &User{Name: "Cadence", Age: 10},
        &User{Name: "Galen", Age: 10},
        &User{Name: "Adams", Age: 10},
	}
}

n, id, err := m.MultiInsert(db, "insert", getMockUserList(), 2)
if err != nil {
    t.Error("insert data failed!", err)
} else {
    t.Logf("multi insert %d rows, last insert id is %d", n, id)
}
```
MultiInsert执行批量插入操作,最后参数2是每批插入数据.
上例中sql是 insert into ...  values (xx, xx, xx, xx), 插入数据有5条,
分批插入,每批2条数据, 实际执行的sql是

|批次|sql|插入参数|
|---|----|---|
|第1批|insert into ...  values (xx, xx, xx, xx), (xx, xx, xx, xx)|user[0], user[1]|
|第2批|insert into ...  values (xx, xx, xx, xx), (xx, xx, xx, xx)|user[2], user[3]|
|第3批|insert into ...  values (xx, xx, xx, xx)|user[4]|

sql的扩展是自动的, 把values后面的括号内容按一批插入数量复制N-1份.

