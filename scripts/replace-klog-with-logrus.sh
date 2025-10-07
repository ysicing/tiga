#!/bin/bash

# 批量替换 klog 为 logrus
set -e

echo "Replacing klog with logrus..."

# 查找所有包含 klog 的 Go 文件
FILES=$(grep -rl '"k8s.io/klog' --include="*.go" .)

for file in $FILES; do
    echo "Processing: $file"

    # 1. 替换 import
    sed -i '' 's|"k8s.io/klog/v2"|"github.com/sirupsen/logrus"|g' "$file"

    # 2. 替换常用函数调用
    sed -i '' 's/klog\.Info(/logrus.Info(/g' "$file"
    sed -i '' 's/klog\.Infof(/logrus.Infof(/g' "$file"
    sed -i '' 's/klog\.Warning(/logrus.Warn(/g' "$file"
    sed -i '' 's/klog\.Warningf(/logrus.Warnf(/g' "$file"
    sed -i '' 's/klog\.Error(/logrus.Error(/g' "$file"
    sed -i '' 's/klog\.Errorf(/logrus.Errorf(/g' "$file"
    sed -i '' 's/klog\.Fatal(/logrus.Fatal(/g' "$file"
    sed -i '' 's/klog\.Fatalf(/logrus.Fatalf(/g' "$file"

    # 3. 替换 V() 详细日志 - 统一改为 Debug
    sed -i '' 's/klog\.V([0-9])\.Info(/logrus.Debug(/g' "$file"
    sed -i '' 's/klog\.V([0-9])\.Infof(/logrus.Debugf(/g' "$file"

    # 4. 其他 klog 特有方法
    sed -i '' 's/klog\.Flush()/\/\/ logrus.Flush() - not needed/g' "$file"
    sed -i '' 's/klog\.InitFlags(nil)/\/\/ klog.InitFlags removed/g' "$file"
done

echo "Replacement complete!"
echo ""
echo "Please review the changes and handle any remaining klog-specific code manually:"
echo "  - klog.V().Enabled() checks"
echo "  - klog custom flags"
echo "  - Any klog-specific features"
