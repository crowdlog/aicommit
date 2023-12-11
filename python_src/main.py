import json
import sys
import os

import tiktoken


def main():
  arg = sys.argv[1]
  enc = tiktoken.get_encoding("cl100k_base")
  try:
    test = enc.encode(loadDiffFile(), )
    print(len(test))
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

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt as e:
        pass