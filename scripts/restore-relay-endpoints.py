#!/usr/bin/env python3
"""
恢复 relay.json 丢失的 20 个端点。

从 git commit 105e30a 提取 Midjourney (16)、Gemini (3)、视频混剪 (1) 端点，
合并到当前 relay.json，处理标签映射。
"""
import json
import subprocess
import sys

RELAY_PATH = 'openapi/relay.json'
OLD_COMMIT = '105e30a'

def main():
    # 1. Load old relay.json from git
    result = subprocess.run(
        ['git', 'show', f'{OLD_COMMIT}:openapi/relay.json'],
        capture_output=True, text=True
    )
    if result.returncode != 0:
        print(f'ERROR: failed to get relay.json from commit {OLD_COMMIT}')
        sys.exit(1)
    old = json.loads(result.stdout)

    # 2. Load current relay.json
    with open(RELAY_PATH, 'r', encoding='utf-8') as f:
        current = json.loads(f.read())

    # 3. Define missing paths and their new tags
    missing_paths = {
        # Midjourney endpoints — all tagged "Midjourney"
        '/mj/submit/imagine': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/describe': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/blend': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/change': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/simple-change': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/action': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/shorten': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/modal': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/edits': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/video': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/submit/upload-discord-images': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/task/{id}/fetch': {'method': 'get', 'new_tag': 'Midjourney'},
        '/mj/task/{id}/image-seed': {'method': 'get', 'new_tag': 'Midjourney'},
        '/mj/task/list-by-condition': {'method': 'post', 'new_tag': 'Midjourney'},
        '/mj/image/{id}': {'method': 'get', 'new_tag': 'Midjourney'},
        '/mj/insight-face/swap': {'method': 'post', 'new_tag': 'Midjourney'},
        # Video remix — change tag from "OpenAI" to "视频生成/Sora兼容格式"
        '/v1/videos/{video_id}/remix': {'method': 'post', 'new_tag': '视频生成/Sora兼容格式'},
        # Gemini extensions — change tag from "Gemini" to "Gemini格式"
        '/v1beta/models/{model}:streamGenerateContent': {'method': 'post', 'new_tag': 'Gemini格式'},
        '/v1beta/models/{model}:embedContent': {'method': 'post', 'new_tag': 'Gemini格式'},
        '/v1beta/models/{model}:countTokens': {'method': 'post', 'new_tag': 'Gemini格式'},
    }

    # 4. Collect all new tag names
    new_tag_names = set()
    for info in missing_paths.values():
        new_tag_names.add(info['new_tag'])

    # 5. Add missing tags to tags array
    current_tag_names = {t['name'] for t in current.get('tags', [])}
    tags_added = []
    for tag_name in sorted(new_tag_names):
        if tag_name not in current_tag_names:
            current['tags'].append({'name': tag_name})
            tags_added.append(tag_name)
            print(f'  + Tag: {tag_name}')

    # 6. Add missing paths
    paths_added = 0
    for path, info in missing_paths.items():
        if path not in current['paths']:
            method = info['method']
            old_path_data = old['paths'].get(path, {}).get(method)
            if old_path_data is None:
                print(f'  WARNING: {method.upper()} {path} not found in old relay.json')
                continue

            # Deep copy to avoid mutating old data
            new_endpoint = json.loads(json.dumps(old_path_data))
            # Update tag
            new_endpoint['tags'] = [info['new_tag']]
            # Add to current
            if path not in current['paths']:
                current['paths'][path] = {}
            current['paths'][path][method] = new_endpoint
            paths_added += 1
            print(f'  + {method.upper()} {path}  [{info["new_tag"]}]')
        else:
            print(f'  SKIP (exists): {path}')

    # 7. Sort paths for consistency
    sorted_paths = dict(sorted(current['paths'].items()))
    current['paths'] = sorted_paths

    # 8. Write
    with open(RELAY_PATH, 'w', encoding='utf-8') as f:
        json.dump(current, f, indent=2, ensure_ascii=False)
        f.write('\n')

    print(f'\n✅ Done: added {tags_added} tags, {paths_added} paths')
    print(f'   relay.json now has {len(current["paths"])} paths')
    if paths_added != 20:
        print(f'   ⚠️ Expected 20, added {paths_added} — check for duplicates')


if __name__ == '__main__':
    main()