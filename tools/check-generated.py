"""重生成前后比较实际产物，兼容首次提交尚无 Git 基线的仓库。"""
import hashlib
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parent.parent


def snapshot():
    return {
        str(path.relative_to(root)): hashlib.sha256(path.read_bytes()).hexdigest()
        for pattern in ("gen/go/**/*.go", "gen/rust/**/rein.hub.v1.rs")
        for path in root.glob(pattern)
    }


before = snapshot()
subprocess.run(["bash", "tools/generate.sh"], cwd=root, check=True)
if before != snapshot():
    raise SystemExit("generated code differs; run bash tools/generate.sh and review the output")
