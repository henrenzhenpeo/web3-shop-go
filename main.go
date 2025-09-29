package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	rpcURL          = "http://127.0.0.1:8545"
	contractAddress = "0x5fbdb2315678afecb367f032d93f642f64180aa3"
	contractABI     = `[
    {
      "inputs": [],
      "stateMutability": "nonpayable",
      "type": "constructor"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "id",
          "type": "uint256"
        }
      ],
      "name": "ProductDisputed",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "id",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "string",
          "name": "name",
          "type": "string"
        },
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "price",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "address",
          "name": "seller",
          "type": "address"
        }
      ],
      "name": "ProductListed",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "id",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "address",
          "name": "buyer",
          "type": "address"
        }
      ],
      "name": "ProductPurchased",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "id",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "address",
          "name": "buyer",
          "type": "address"
        }
      ],
      "name": "ProductRefunded",
      "type": "event"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "_id",
          "type": "uint256"
        }
      ],
      "name": "disputeProduct",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "string",
          "name": "_name",
          "type": "string"
        },
        {
          "internalType": "string",
          "name": "_description",
          "type": "string"
        },
        {
          "internalType": "uint256",
          "name": "_price",
          "type": "uint256"
        }
      ],
      "name": "listProduct",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "owner",
      "outputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "productCount",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "name": "products",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "id",
          "type": "uint256"
        },
        {
          "internalType": "address payable",
          "name": "seller",
          "type": "address"
        },
        {
          "internalType": "string",
          "name": "name",
          "type": "string"
        },
        {
          "internalType": "string",
          "name": "description",
          "type": "string"
        },
        {
          "internalType": "uint256",
          "name": "price",
          "type": "uint256"
        },
        {
          "internalType": "enum Ecommerce.ProductStatus",
          "name": "status",
          "type": "uint8"
        },
        {
          "internalType": "address",
          "name": "buyer",
          "type": "address"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "_id",
          "type": "uint256"
        }
      ],
      "name": "purchaseProduct",
      "outputs": [],
      "stateMutability": "payable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "_id",
          "type": "uint256"
        },
        {
          "internalType": "bool",
          "name": "refund",
          "type": "bool"
        }
      ],
      "name": "resolveDispute",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ]`
	privateKeyHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

var (
	client    *ethclient.Client
	parsedABI abi.ABI
	fromKey   *ecdsa.PrivateKey
	fromAddr  common.Address
	contract  common.Address
)

type Product struct {
	ID          int    `json:"id"`
	Seller      string `json:"seller"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
	Status      uint8  `json:"status"`
	Buyer       string `json:"buyer"`
}

// 必须和 Solidity Product struct 的顺序保持一致
type ProductRaw struct {
	ID          *big.Int
	Seller      common.Address
	Name        string
	Description string
	Price       *big.Int
	Status      uint8
	Buyer       common.Address
}

func main() {
	var err error
	client, err = ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect Ethereum client: %v", err)
	}

	parsedABI, err = abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		log.Fatalf("Failed to parse ABI: %v", err)
	}

	fromKey, err = crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	fromAddr = crypto.PubkeyToAddress(fromKey.PublicKey)
	contract = common.HexToAddress(contractAddress)

	http.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		productsHandler(w, r)
	})

	http.HandleFunc("/purchase", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		purchaseHandler(w, r)
	})

	fmt.Println("✅ Server running at http://localhost:9080")
	log.Fatal(http.ListenAndServe(":9080", nil))
}

// 查询合约中的商品

func productsHandler(w http.ResponseWriter, r *http.Request) {
	// 调用 productCount()
	data, err := parsedABI.Pack("productCount")
	if err != nil {
		log.Printf("Failed to pack productCount: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	msg := ethereum.CallMsg{
		From: fromAddr,
		To:   &contract,
		Data: data,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Printf("Failed to call productCount: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 解析返回值
	count, err := parsedABI.Unpack("productCount", result)
	if err != nil || len(count) == 0 {
		log.Printf("Failed to unpack productCount: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	total := count[0].(*big.Int).Int64()
	if total == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Product{})
		return
	}

	products := make([]Product, 0)

	// 遍历 products(i)
	for i := int64(1); i <= total; i++ {
		data, err := parsedABI.Pack("products", big.NewInt(i))
		if err != nil {
			log.Printf("Failed to pack products(%d): %v", i, err)
			continue
		}

		msg := ethereum.CallMsg{
			From: fromAddr,
			To:   &contract,
			Data: data,
		}

		res, err := client.CallContract(context.Background(), msg, nil)
		if err != nil {
			log.Printf("Failed to call products(%d): %v", i, err)
			continue
		}

		if len(res) == 0 {
			// 该商品不存在
			continue
		}

		// 解码 tuple
		out, err := parsedABI.Unpack("products", res)
		if err != nil {
			log.Printf("Failed to unpack product %d: %v", i, err)
			continue
		}
		if len(out) != 7 {
			log.Printf("Unexpected output length for product %d: %d", i, len(out))
			continue
		}

		p := Product{
			ID:          int(out[0].(*big.Int).Int64()),
			Seller:      out[1].(common.Address).Hex(),
			Name:        out[2].(string),
			Description: out[3].(string),
			Price:       out[4].(*big.Int).String(),
			Status:      out[5].(uint8),
			Buyer:       out[6].(common.Address).Hex(),
		}

		products = append(products, p)
	}

	// 返回 JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// 购买商品
func purchaseHandler(w http.ResponseWriter, r *http.Request) {
	// ⚡ 处理 CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("⚡ Incoming /purchase request from %s", r.RemoteAddr)

	type Req struct {
		ProductID int    `json:"productId"`
		Amount    string `json:"amount"` // 已经是 wei
	}

	var req Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid request body: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("Purchase request: ProductID=%d, Amount=%s", req.ProductID, req.Amount)

	// 直接使用前端传过来的 wei
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		log.Printf("❌ Invalid amount: %s", req.Amount)
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	// 打包调用数据
	data, err := parsedABI.Pack("purchaseProduct", big.NewInt(int64(req.ProductID)))
	if err != nil {
		log.Printf("❌ Pack purchaseProduct failed: %v", err)
		http.Error(w, "pack tx data failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取 nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		log.Printf("❌ Get nonce failed: %v", err)
		http.Error(w, "get nonce failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Printf("❌ SuggestGasPrice failed: %v", err)
		http.Error(w, "get gas price failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	msg := ethereum.CallMsg{From: fromAddr, To: &contract, Value: amount, Data: data}
	gasLimit, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		log.Printf("⚠️ EstimateGas failed, fallback 1000000: %v", err)
		gasLimit = 1_000_000
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Printf("❌ Get chainID failed: %v", err)
		http.Error(w, "get chain id failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tx := types.NewTransaction(nonce, contract, amount, gasLimit, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), fromKey)
	if err != nil {
		log.Printf("❌ SignTx failed: %v", err)
		http.Error(w, "sign tx failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := client.SendTransaction(context.Background(), signedTx); err != nil {
		log.Printf("❌ SendTransaction failed: %v", err)
		http.Error(w, "send tx failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Transaction sent: %s", signedTx.Hash().Hex())

	// ⚡ 等待交易上链回执（可选）
	/*
	   receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	   if err != nil {
	       log.Printf("❌ WaitMined failed: %v", err)
	   } else {
	       log.Printf("✅ Transaction mined, status=%d", receipt.Status)
	   }
	*/

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"txHash": signedTx.Hash().Hex(),
	})
}
