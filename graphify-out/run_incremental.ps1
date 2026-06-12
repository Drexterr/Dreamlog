$py = Get-Content graphify-out\.graphify_python
& $py -c "import sys, json; from graphify.detect import detect_incremental, save_manifest; from pathlib import Path; result = detect_incremental(Path('.')); Path('graphify-out/.graphify_incremental.json').write_text(json.dumps(result, ensure_ascii=False), encoding='utf-8')"
