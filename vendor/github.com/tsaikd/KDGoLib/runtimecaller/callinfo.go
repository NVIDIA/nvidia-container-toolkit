package runtimecaller

import "runtime"

// CallInfo contains runtime caller information
type CallInfo interface {
	// builtin data
	PC() uintptr
	FilePath() string
	Line() int

	// extra info after some process
	PCFunc() *runtime.Func
	PackageName() string
	FileDir() string
	FileName() string
	FuncName() string
}

// CallInfoImpl implement CallInfo
type CallInfoImpl struct {
	// builtin data
	pc       uintptr
	filePath string
	line     int

	// extra info after some process
	pcFunc      *runtime.Func
	packageName string
	fileDir     string
	fileName    string
	funcName    string
}

// PC return CallInfo data
func (t CallInfoImpl) PC() uintptr {
	return t.pc
}

// FilePath return CallInfo data
func (t CallInfoImpl) FilePath() string {
	return t.filePath
}

// Line return CallInfo data
func (t CallInfoImpl) Line() int {
	return t.line
}

// PCFunc return CallInfo data
func (t CallInfoImpl) PCFunc() *runtime.Func {
	return t.pcFunc
}

// PackageName return CallInfo data
func (t CallInfoImpl) PackageName() string {
	return t.packageName
}

// FileDir return CallInfo data
func (t CallInfoImpl) FileDir() string {
	return t.fileDir
}

// FileName return CallInfo data
func (t CallInfoImpl) FileName() string {
	return t.fileName
}

// FuncName return CallInfo data
func (t CallInfoImpl) FuncName() string {
	return t.funcName
}

var _ = CallInfo(CallInfoImpl{})
