#!/usr/bin/env python3
"""Minimal YAML-to-JSON converter for the seed scripts.

Usage:  python3 scripts/_parse-yaml.py seed/minio.yml
        python3 scripts/_parse-yaml.py seed/data.yml
"""
import sys, json, re

def parse_simple_yaml(path):
    """
    Parses the specific seed YAML structures without PyYAML.
    Handles: top-level scalars, nested maps (2-space indent), lists of maps.
    Returns a nested dict/list structure.
    """
    with open(path) as f:
        lines = f.readlines()

    root = {}
    stack = [(root, -1)]  # (container, indent_level)

    list_item_indent = {}  # track which indents are list-item indents

    def current_parent(ind):
        while len(stack) > 1 and stack[-1][1] >= ind:
            stack.pop()
        return stack[-1][0]

    def parse_scalar(s):
        s = s.strip().strip('"').strip("'")
        if s.lower() == 'true':  return True
        if s.lower() == 'false': return False
        if s.lower() in ('null', '~', ''): return None
        try: return int(s)
        except ValueError: pass
        try: return float(s)
        except ValueError: pass
        return s

    prev_key = {}  # indent -> key for the current block

    for raw_line in lines:
        line = raw_line.rstrip()
        stripped = line.lstrip()
        if not stripped or stripped.startswith('#'):
            continue
        indent = len(line) - len(stripped)

        # List item
        if stripped.startswith('- '):
            rest = stripped[2:]
            parent = current_parent(indent)

            # Ensure parent knows this key holds a list
            if isinstance(parent, dict):
                key = prev_key.get(indent - 2) or prev_key.get(indent)
                if key and not isinstance(parent.get(key), list):
                    parent[key] = []
                container = parent.get(key, []) if key else parent
                if not isinstance(container, list):
                    container = []
                    if key: parent[key] = container
            else:
                container = parent

            if isinstance(container, list):
                m = re.match(r'^(\w[\w\s]*?):\s*(.*)', rest)
                if m:
                    new_item = {m.group(1): parse_scalar(m.group(2))}
                    container.append(new_item)
                    stack.append((new_item, indent))
                    prev_key[indent] = m.group(1)
                else:
                    container.append(parse_scalar(rest))
            continue

        # Key: value
        m = re.match(r'^([\w][\w\s]*?):\s*(.*)', stripped)
        if m:
            key = m.group(1).strip()
            val_str = m.group(2).strip()

            parent = current_parent(indent)
            prev_key[indent] = key

            if not val_str:
                # Next lines are nested content
                parent[key] = {}
                stack.append((parent[key], indent))
            else:
                parent[key] = parse_scalar(val_str)

    return root

try:
    import yaml
    with open(sys.argv[1]) as f:
        data = yaml.safe_load(f)
    print(json.dumps(data))
except ImportError:
    data = parse_simple_yaml(sys.argv[1])
    print(json.dumps(data))
