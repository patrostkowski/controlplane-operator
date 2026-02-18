// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type ZapLogger struct {
	log logr.Logger
}

func NewZapLogger(logLevel Level) *ZapLogger {
	newZap := zap.New(zap.UseDevMode(true), zap.Level(logLevel.toZapLevel()))
	return &ZapLogger{
		log: newZap,
	}
}

func (l Level) toZapLevel() zapcore.Level {
	switch l {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func (l *ZapLogger) GetBaseLogrInstance() logr.Logger {
	return l.log
}

func (l *ZapLogger) Debug(msg string, kv ...any) {
	l.log.V(int(zapcore.DebugLevel)).Info(msg, kv...)
}

func (l *ZapLogger) Info(msg string, kv ...any) {
	l.log.V(int(zapcore.InfoLevel)).Info(msg, kv...)
}

func (l *ZapLogger) Warn(msg string, kv ...any) {
	l.log.V(int(zapcore.WarnLevel)).Info(msg, kv...)
}
func (l *ZapLogger) Error(err error, msg string, kv ...any) {
	l.log.V(int(zapcore.ErrorLevel)).Error(err, msg, kv...)
}
