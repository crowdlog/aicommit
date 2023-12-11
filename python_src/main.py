import json
import sys
import os

import tiktoken
# import sqlite3
from concurrent.futures import ThreadPoolExecutor
import re

from typing import List, TypedDict, cast

class FileDiff(TypedDict, total=False):
    token_count: int
    file_diff: str
    file_name: str

def main():
  try:
    test = split_gitdiff_by_token_limit(loadDiffFile(), 400)
    json_str = json.dumps(test, indent=4)
    print(json_str)
    cwd = os.getcwd()
    print(cwd)
  except Exception as e:
    print(e)

        # args = json.loads(args)
        # opts = args["opts"]
        # print(opts)

        # result = ''

        # result = json.dumps(result)

        # sys.stdout.write(result + "\n")
        # sys.stdout.flush()

def loadDiffFile ():
    diffFile = open("diff.diff", "r")
    diff = diffFile.read()
    diffFile.close()
    return diff


def count_tokens_for_file_diff(file_diff: FileDiff, encoder: tiktoken.Encoding):
    file_diff['token_count'] = len(encoder.encode(file_diff['file_diff']))
    return file_diff
  
def count_tokens_on_string(string, encoder: tiktoken.Encoding):
    return len(encoder.encode(string))
  
def getFileExtension (file_diff):
    return file_diff.split("b/")[1].split(".")[1]
  
def get_file_diffs(diff_text) -> List[FileDiff]:
  # Splitting the diff text into individual file diffs
  file_diffs = diff_text.split('\ndiff --git')
  pattern = r'^\+\+\+ [ab]/([a-zA-Z0-9_./-]+\.[a-zA-Z0-9_]+)'

  results: List[FileDiff] = []
  for diff_block in file_diffs:
      if diff_block.strip():
          # Adding 'diff --git' back to the diff block, except for the first split which already lacks it
          diff_block_with_prefix = 'diff --git' + diff_block if diff_block != file_diffs[0] else diff_block

          # Using a more specific regex pattern to match file paths only in the diff header lines
          match = re.search(pattern, diff_block_with_prefix, re.MULTILINE)
          if match:
              file_name = match.group(1)
              extension = file_name.split('.')[-1]
          else:
              file_name = "unknown"
              extension = "unknown"
          file_diff = cast(FileDiff, {'extension': f'.{extension}', 'file_name': file_name, 'file_diff': diff_block_with_prefix})
          results.append(file_diff)
  return results

def split_large_file_diff(file_diff: FileDiff, token_limit, encoder) -> List[FileDiff]:
    lines = file_diff['file_diff'].split('\n')
    with ThreadPoolExecutor() as executor:
        token_counts = list(executor.map(lambda fd: count_tokens_on_string(fd, encoder), lines))
    
    current_chunk: List[str] = []
    chunks: List[FileDiff] = []
    current_token_count = 0

    for i, line in enumerate(lines):
        if current_token_count + token_counts[i] > token_limit:
            if current_chunk:  # Check if there's anything in the current chunk
                file_diff_chunk = cast(FileDiff, {'file_diff': '\n'.join(current_chunk), 'file_name': file_diff['file_name'], 'token_count': current_token_count})
                chunks.append(file_diff_chunk)
            current_chunk = [line]
            current_token_count = token_counts[i]
        else:
            current_chunk.append(line)
            current_token_count += token_counts[i]

    if current_chunk:
        file_diff_chunk = cast(FileDiff, {'file_diff': '\n'.join(current_chunk), 'file_name': file_diff['file_name'], 'token_count': current_token_count})
        chunks.append(file_diff_chunk)

    return chunks

def split_gitdiff_by_token_limit(diff_string: str, token_limit: int):
    # Get the encoding
    enc = tiktoken.get_encoding("cl100k_base")

    # Split the diff string into separate file diffs
    file_diffs = get_file_diffs(diff_string)

    # Parallel processing to count tokens
    with ThreadPoolExecutor() as executor:
        file_diffs_with_token_counts = list(executor.map(lambda fd: count_tokens_for_file_diff(fd, enc), file_diffs))
    # Initialize variables
    result_groups: List[List[FileDiff]] = []
    current_group: List[FileDiff] = []
    current_token_count = 0

    for i, file_diff in enumerate(file_diffs_with_token_counts):
        # If the file diff is too large, split it into smaller chunks
        if file_diff['token_count'] > token_limit:
            file_diff_chunks = split_large_file_diff(file_diff, token_limit, enc)
            for file_diff_chunk in file_diff_chunks:
                current_group.append(file_diff_chunk)
                result_groups.append(current_group)
                current_group = []
            continue
        # Check if adding this file diff exceeds the token limit
        elif current_token_count + file_diffs_with_token_counts[i]['token_count'] > token_limit:
            # If the current group is not empty, add it to the result
            if current_group:
                result_groups.append(current_group)
                current_group = []

            # Reset the token count
            current_token_count = 0

        # Add the current file diff to the group and update the token count
        current_group.append(file_diff)
        current_token_count += file_diffs_with_token_counts[i]['token_count']

    # Add the last group to the result if it's not empty
    if current_group:
        result_groups.append(current_group)

    return result_groups



if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt as e:
        pass