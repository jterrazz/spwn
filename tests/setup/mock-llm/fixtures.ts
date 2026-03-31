/** Simple greeting response. */
export const HELLO_RESPONSE = "Hello from mock LLM!";

/** Response listing workspace files. */
export const FILE_LIST_RESPONSE =
  "Here are the files in the workspace: README.md, main.go";

/** Response for a code generation prompt. */
export const CODE_GENERATION_RESPONSE = `Here is the code:
\`\`\`go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
\`\`\``;

/** Empty / minimal response. */
export const EMPTY_RESPONSE = "";

/** Error-like response (the LLM says it cannot do something). */
export const REFUSAL_RESPONSE =
  "I'm sorry, I cannot perform that action in this environment.";
