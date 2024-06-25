# sticker ssz

> it does what it says on the sticker

SSZ using reflection

## Example

```go
import (
	"github.com/ethereum/go-ethereum/ssz"
)

// Create types with ssz field tags where needed

type Checkpoint struct {
    Epoch uint64
    Root []byte `ssz: "Vector,32"`
}

type AttestationData struct {
    Slot uint64
    Index uint64
    BeaconBlockRoot []byte `ssz: "Vector,32"`
    Source Checkpoint
    Target Checkpoint
}

// Create an ssz type from a struct
AttestationDataType, _ := ssz.MakeGenType(AttestationData{}, nil)

// ssz types have all the methods you know and love
d := AttestationDataType.Default()
AttestationDataType.deserialize(AttestationDataType.Serialize(d))
AttestationDataType.HashTreeRoot(d)
```

## License

MIT