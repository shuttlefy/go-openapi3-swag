# 通用代码规范（语言无关）

---

## 1. 函数职责

**规则**：单个函数只做一件事，超过 40 行需审查是否可拆分。

```
// ✅ 各自独立：validate() / marshal() / merge()
// ❌ 一个函数包含校验、序列化、写文件三件事
```

---

## 2. 注释

**规则**：注释解释"为什么"，而非"做什么"；不写叙述性注释。

```
// ✅ // $ref takes precedence per OAS spec: ref-only objects must not contain other fields
// ❌ // marshal the struct to JSON
// ❌ // increment counter
```

---

## 3. 魔法数字 / 魔法字符串

**规则**：使用具名常量替代字面量。

```
// ✅ const maxRetries = 3
// ❌ if retries > 3 { ... }
```

---

## 4. 测试覆盖

**规则**：新功能必须有测试；边界条件（nil、空、最大值）要覆盖；测试名应描述场景。

```
// ✅ TestOrderedMap_Delete_LastElement
// ❌ TestDelete
```

---

## 5. 死代码

**规则**：无法到达的代码、注释掉的旧代码、未使用的变量必须删除。

```
// ❌ _ = first   （无意义的空赋值）
// ❌ // old code: data.init()
```

---

## 扩展此文件

在末尾追加新规则节，保持编号连续。
