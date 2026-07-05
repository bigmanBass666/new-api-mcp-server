import json

with open('D:/Test/new-api-mcp-server/openapi/api.new.json', 'r') as f:
    spec = json.load(f)

schemas = spec['components']['schemas']

# --- Fix 1: Channel schema ---
if 'Channel' in schemas:
    ch = schemas['Channel']
    # Fix groups -> group
    if 'groups' in ch['properties']:
        ch['properties']['group'] = ch['properties'].pop('groups')
        ch['properties']['group']['description'] = 'Channel group (e.g., default, vip)'

    missing = {
        'key': {'type': 'string', 'x-omit': 'true', 'description': 'API key for this channel'},
        'channel_info': {'type': 'object', 'properties': {
            'is_multi_key': {'type': 'boolean'},
            'multi_key_size': {'type': 'integer'},
            'multi_key_polling_index': {'type': 'integer'},
            'multi_key_mode': {'type': 'string', 'enum': ['random', 'polling']}
        }, 'description': 'Multi-key rotation configuration'},
        'setting': {'type': 'string', 'description': 'Channel settings JSON'},
        'settings': {'type': 'string', 'description': 'Extended settings JSON'},
        'model_mapping': {'type': 'string', 'nullable': True},
        'param_override': {'type': 'string', 'nullable': True},
        'header_override': {'type': 'string', 'nullable': True},
        'auto_ban': {'type': 'integer', 'description': 'Auto-ban on failure'},
        'used_quota': {'type': 'integer', 'description': 'Used quota'},
        'balance': {'type': 'number', 'description': 'Account balance'},
        'balance_updated_time': {'type': 'integer'},
        'created_time': {'type': 'integer'},
        'test_time': {'type': 'integer'},
        'response_time': {'type': 'integer', 'description': 'Response time in ms'},
        'other': {'type': 'string'},
        'other_info': {'type': 'string'},
        'remark': {'type': 'string', 'nullable': True},
        'openai_organization': {'type': 'string'},
        'test_model': {'type': 'string', 'nullable': True}
    }
    for fn, fs in missing.items():
        if fn not in ch['properties']:
            ch['properties'][fn] = fs
            print(f'Added {fn} to Channel')
    ch['required'] = ['name', 'type']
    print(f'Channel now has {len(ch["properties"])} properties')

# --- Fix 2: Token schema ---
if 'Token' in schemas:
    tk = schemas['Token']
    missing = {
        'group': {'type': 'string'},
        'used_quota': {'type': 'integer'},
        'model_limits_enabled': {'type': 'boolean'},
        'model_limits': {'type': 'string'},
        'allow_ips': {'type': 'string'},
        'cross_group_retry': {'type': 'boolean'},
        'created_time': {'type': 'integer'},
        'accessed_time': {'type': 'integer'}
    }
    for fn, fs in missing.items():
        if fn not in tk['properties']:
            tk['properties'][fn] = fs
            print(f'Added {fn} to Token')

# --- Fix 3: User schema ---
if 'User' in schemas:
    us = schemas['User']
    missing = {
        'aff_code': {'type': 'string'},
        'created_at': {'type': 'integer'},
        'last_login_at': {'type': 'integer'},
        'setting': {'type': 'string'}
    }
    for fn, fs in missing.items():
        if fn not in us['properties']:
            us['properties'][fn] = fs
            print(f'Added {fn} to User')

# --- Fix 4: Log schema ---
if 'Log' in schemas:
    lg = schemas['Log']
    missing = {
        'username': {'type': 'string'},
        'token_name': {'type': 'string'},
        'model_name': {'type': 'string'},
        'quota': {'type': 'integer'},
        'prompt_tokens': {'type': 'integer'},
        'completion_tokens': {'type': 'integer'},
        'use_time': {'type': 'number'},
        'is_stream': {'type': 'boolean'},
        'channel': {'type': 'integer'},
        'channel_name': {'type': 'string'},
        'token_id': {'type': 'integer'},
        'group': {'type': 'string'},
        'request_id': {'type': 'string'},
        'upstream_request_id': {'type': 'string'}
    }
    for fn, fs in missing.items():
        if fn not in lg['properties']:
            lg['properties'][fn] = fs
            print(f'Added {fn} to Log')

# Write output
outpath = 'D:/Test/new-api-mcp-server/openapi/api.new.json'
with open(outpath, 'w') as f:
    json.dump(spec, f, indent=2, ensure_ascii=False)

print(f'\nWritten to {outpath}')
print('Done!')