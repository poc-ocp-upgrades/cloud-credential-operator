package v1

import (
	"bytes"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func NewScheme() (*runtime.Scheme, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return SchemeBuilder.Build()
}

type ProviderCodec struct {
	encoder	runtime.Encoder
	decoder	runtime.Decoder
}

func NewCodec() (*ProviderCodec, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	scheme, err := NewScheme()
	if err != nil {
		return nil, err
	}
	codecFactory := serializer.NewCodecFactory(scheme)
	encoder, err := newEncoder(&codecFactory)
	if err != nil {
		return nil, err
	}
	codec := ProviderCodec{encoder: encoder, decoder: codecFactory.UniversalDecoder(SchemeGroupVersion)}
	return &codec, nil
}
func (codec *ProviderCodec) EncodeProviderSpec(in runtime.Object) (*runtime.RawExtension, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var buf bytes.Buffer
	if err := codec.encoder.Encode(in, &buf); err != nil {
		return nil, fmt.Errorf("encoding failed: %v", err)
	}
	return &runtime.RawExtension{Raw: buf.Bytes()}, nil
}
func (codec *ProviderCodec) DecodeProviderSpec(providerConfig *runtime.RawExtension, out runtime.Object) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_, _, err := codec.decoder.Decode(providerConfig.Raw, nil, out)
	if err != nil {
		return fmt.Errorf("decoding failure: %v", err)
	}
	return nil
}
func (codec *ProviderCodec) EncodeProviderStatus(in runtime.Object) (*runtime.RawExtension, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var buf bytes.Buffer
	if err := codec.encoder.Encode(in, &buf); err != nil {
		return nil, fmt.Errorf("encoding failed: %v", err)
	}
	return &runtime.RawExtension{Raw: buf.Bytes()}, nil
}
func (codec *ProviderCodec) DecodeProviderStatus(providerStatus *runtime.RawExtension, out runtime.Object) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if providerStatus != nil {
		_, _, err := codec.decoder.Decode(providerStatus.Raw, nil, out)
		if err != nil {
			return fmt.Errorf("decoding failure: %v", err)
		}
		return nil
	}
	return nil
}
func newEncoder(codecFactory *serializer.CodecFactory) (runtime.Encoder, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	serializerInfos := codecFactory.SupportedMediaTypes()
	if len(serializerInfos) == 0 {
		return nil, fmt.Errorf("unable to find any serlializers")
	}
	encoder := codecFactory.EncoderForVersion(serializerInfos[0].Serializer, SchemeGroupVersion)
	return encoder, nil
}
