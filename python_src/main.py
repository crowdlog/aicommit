import json
import tiktoken
from concurrent.futures import ThreadPoolExecutor
import re

from typing import List, TypedDict, cast
import sqlite3
from concurrent.futures import ThreadPoolExecutor, as_completed

class FileDiff(TypedDict, total=False):
    token_count: int
    file_diff: str
    file_name: str


openai_models = {
    "gpt-4-1106-preview": 128000,
    "gpt-4-vision-preview": 128000,
    "gpt-4": 8192,
    "gpt-4-32k": 32768,
    "gpt-4-0613": 8192,
    "gpt-4-32k-0613": 32768,
    "gpt-4-0314": 8192,
    "gpt-4-32k-0314": 32768,
    "gpt-3.5-turbo-1106": 16385,
    "gpt-3.5-turbo": 4096,
    "gpt-3.5-turbo-16k": 16385,
    "gpt-3.5-turbo-instruct": 4096,
    "gpt-3.5-turbo-0613": 4096,
    "gpt-3.5-turbo-16k-0613": 16385,
    "gpt-3.5-turbo-0301": 4096,
    "text-davinci-003": 4096,
    "text-davinci-002": 4096,
    "code-davinci-002": 8001,
}

def main():
    try:
                # Connect to the SQLite database
        conn = sqlite3.connect('aicommit.db')
        cursor = conn.cursor()

        # Query to select the row with the ID 'diff'
        query = "SELECT * FROM diff WHERE id = 'diff'"
        cursor.execute(query)

        # Fetch the row
        row = cursor.fetchone()

        # Close the connection

        # Convert the row to a dictionary if it exists
        if row:
            row_dict = {
                'id': row[0],
                'diff': row[1],
                'date_created': row[2],
                'diff_structured_json': row[3],
                'model': row[4],
                'ai_provider': row[5],
                'prompts': json.loads(row[6])  # Deserialize the JSON string
            }
        else:
            row_dict = {}

        split_gitdiff_args = SplitGitDiffArgs(
            diff_string=row_dict['diff'],
            model=row_dict['model'],
            prompts=row_dict['prompts']
        )
        test = split_gitdiff(split_gitdiff_args)
        json_str = json.dumps(test, indent=4)

        update_query = "UPDATE diff SET diff_structured_json = ? WHERE id = 'diff'"
        cursor.execute(update_query, (json_str,))
        conn.commit()

        conn.close()
    except Exception as e:
        print(e)

        # args = json.loads(args)
        # opts = args["opts"]
        # print(opts)

        # result = ''

        # result = json.dumps(result)

        # sys.stdout.write(result + "\n")
        # sys.stdout.flush()


def loadDiffFile():
    diffFile = open("diff.diff", "r")
    diff = diffFile.read()
    diffFile.close()
    return diff


def count_tokens_for_file_diff(file_diff: FileDiff, encoder: tiktoken.Encoding):
    file_diff["token_count"] = len(encoder.encode(file_diff["file_diff"]))
    return file_diff


def count_tokens_on_string(string, encoder: tiktoken.Encoding):
    return len(encoder.encode(string))


def getFileExtension(file_diff):
    return file_diff.split("b/")[1].split(".")[1]


def get_file_diffs(diff_text) -> List[FileDiff]:
    # Splitting the diff text into individual file diffs
    file_diffs = diff_text.split("\ndiff --git")
    pattern = r"^\+\+\+ [ab]/([a-zA-Z0-9_./-]+\.[a-zA-Z0-9_]+)"

    results: List[FileDiff] = []
    for diff_block in file_diffs:
        if diff_block.strip():
            # Adding 'diff --git' back to the diff block, except for the first split which already lacks it
            diff_block_with_prefix = (
                "diff --git" + diff_block if diff_block != file_diffs[0] else diff_block
            )

            # Using a more specific regex pattern to match file paths only in the diff header lines
            match = re.search(pattern, diff_block_with_prefix, re.MULTILINE)
            if match:
                file_name = match.group(1)
                extension = file_name.split(".")[-1]
            else:
                file_name = "unknown"
                extension = "unknown"
            file_diff = cast(
                FileDiff,
                {
                    "extension": f".{extension}",
                    "file_name": file_name,
                    "file_diff": diff_block_with_prefix,
                },
            )
            results.append(file_diff)
    return results


def split_large_file_diff(file_diff: FileDiff, token_limit, encoder) -> List[FileDiff]:
    lines = file_diff["file_diff"].split("\n")
    with ThreadPoolExecutor() as executor:
        token_counts = list(
            executor.map(lambda fd: count_tokens_on_string(fd, encoder), lines)
        )

    current_chunk: List[str] = []
    chunks: List[FileDiff] = []
    current_token_count = 0

    for i, line in enumerate(lines):
        if current_token_count + token_counts[i] > token_limit:
            if current_chunk:  # Check if there's anything in the current chunk
                file_diff_chunk = cast(
                    FileDiff,
                    {
                        "file_diff": "\n".join(current_chunk),
                        "file_name": file_diff["file_name"],
                        "token_count": current_token_count,
                    },
                )
                chunks.append(file_diff_chunk)
            current_chunk = [line]
            current_token_count = token_counts[i]
        else:
            current_chunk.append(line)
            current_token_count += token_counts[i]

    if current_chunk:
        file_diff_chunk = cast(
            FileDiff,
            {
                "file_diff": "\n".join(current_chunk),
                "file_name": file_diff["file_name"],
                "token_count": current_token_count,
            },
        )
        chunks.append(file_diff_chunk)

    return chunks

class SplitGitDiffArgs(TypedDict):
    diff_string: str
    model: str
    prompts: List[str]

def split_gitdiff(args: SplitGitDiffArgs):
    # Get the encoding
    enc = tiktoken.encoding_for_model(args['model'])

    # Split the diff string into separate file diffs
    file_diffs = get_file_diffs(args['diff_string'])

    with ThreadPoolExecutor() as executor:
        # Schedule tasks for count_tokens_for_file_diff
        future_to_file_diff = {executor.submit(count_tokens_for_file_diff, fd, enc): fd for fd in file_diffs}
        
        # Schedule tasks for count_tokens_on_string for each prompt
        future_to_prompt = {executor.submit(count_tokens_on_string, prompt, enc): prompt for prompt in args['prompts']}

        # Prepare to store results
        prompts_token_counts = []
        file_diffs_with_token_counts = []

        # Gather all future objects
        all_futures = list(future_to_file_diff.keys()) + list(future_to_prompt.keys())

        # Process results as they complete
        for future in as_completed(all_futures):
            try:
                result = future.result()
                if future in future_to_file_diff:
                    # This result is from count_tokens_for_file_diff
                    file_diffs_with_token_counts.append(result)
                elif future in future_to_prompt:
                    # This result is from count_tokens_on_string
                    prompts_token_counts.append(result)
            except Exception as e:
                # Handle exceptions
                print(f"An error occurred: {e}")

    token_limit = openai_models[args['model']] - sum(prompts_token_counts)

    # Initialize variables
    result_groups: List[List[FileDiff]] = []
    current_group: List[FileDiff] = []
    current_token_count = 0

    for i, file_diff in enumerate(file_diffs_with_token_counts):
        # If the file diff is too large, split it into smaller chunks
        if file_diff["token_count"] > token_limit:
            file_diff_chunks = split_large_file_diff(file_diff, token_limit, enc)
            for file_diff_chunk in file_diff_chunks:
                current_group.append(file_diff_chunk)
                result_groups.append(current_group)
                current_group = []
            continue
        # Check if adding this file diff exceeds the token limit
        elif (
            current_token_count + file_diffs_with_token_counts[i]["token_count"]
            > token_limit
        ):
            # If the current group is not empty, add it to the result
            if current_group:
                result_groups.append(current_group)
                current_group = []

            # Reset the token count
            current_token_count = 0

        # Add the current file diff to the group and update the token count
        current_group.append(file_diff)
        current_token_count += file_diffs_with_token_counts[i]["token_count"]

    # Add the last group to the result if it's not empty
    if current_group:
        result_groups.append(current_group)

    return result_groups


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt as e:
        pass
