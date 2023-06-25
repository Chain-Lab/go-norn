# Karmem 数据结构

[Karmem](https://github.com/inkeliz/karmem) 是 Github 上开源的一个快速的数据序列化库，在 go-chronos 中主要用于 P2P 下区块、交易的发送，以及在整个系统内相关数据结构的表示

Karmem 可以将符合格式的 `.km` 文件编译到对应语言的源文件，目前支持的有：

- AssemblyScript 0.20.16
- C/Emscripten
- C#/.NET 7
- Golang 1.19/TinyGo 0.25.0
- Odin
- Swift 5.7/SwiftWasm 5.7
- Zig 0.10

编译为 golang 下的代码可以使用命令

```shell
karmem build --golang -o "km" core.km
```

## 核心数据结构

文件 `core.km` 下定义了一系列的数据结构，可以生成golang下对应的结构体

```
karmem structs @packed(true) @golang.package(`core`);

// 交易体结构
struct TransactionBody table {
    Hash [32]byte;					// 交易的哈希值
    Signature [<73]byte;			// 交易的签名
    Address [20]byte;				// 交易发送者的地址
    Public [33]byte;				// 交易发送者的公钥
    Data []byte;					// 可选数据段，可以被用来调用智能合约
    Expire int64;					// 交易过期时间
    Timestamp int64;				// 交易的时间戳
}

// 交易，只包含一个body
struct Transaction inline {			// karmem 下不支持嵌套，只能使用 inline 的结构来实现嵌套的功能
    Body TransactionBody;			// 交易体
}

// 创世参数信息
struct GenesisParams table {
    Order [128]byte;				// VDF 计算所在群的阶
    TimeParam int64;				// VDF 计算时间参数
    Seed [32]byte;					// VDF 初始 seed
    VerifyParam [32]byte;			// VDF 验证使用参数
}

// 普通区块参数
struct GeneralParams table {
    Result []byte;					// VDF 计算结果，如果还没有计算出来，使用原有的结果
    Proof []byte;					// VDF 计算证明
    RandomNumber [33]byte;			// VRF 生成的随机数
    S []byte;						// VRF 计算证明(s, t)
    T []byte;
}

// 区块头结构
struct BlockHeader table {
    Timestamp int64;				// 区块生成时间戳
    PrevBlockHash [32]byte;			// 前一个区块的哈希值
    BlockHash [32]byte;				// 当前区块的哈希值
    MerkleRoot [32]byte;			// 交易的 Merkle 树哈希值
    Height int64;					// 当前区块的高度
    PublicKey [33]byte;				// 当前区块生成者的公钥
    Params []byte;					// 区块附带的参数信息，创世参数或普通参数
}

// 区块所的结构
struct Block table {
    Header BlockHeader;				// 区块头
    Transactions []Transaction;		// 交易列表
}
```

