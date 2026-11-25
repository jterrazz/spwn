---
kind: guide
line: "Every form the compiler reads, once."
---

# Forms

Not a corpus page: each form of the tag grammar, once, beside the ones the corpus pages use.

<Table id="carried-today" of="conviction" in={ref("blueprint/philosophy")}
where={query({ carried_in: { has: state("shipped") } })} columns={["says", "carried_in", "@weight"]} sort="says" />
<Page id="forms/child" face="open" />
<Page id="forms/other" />

A line of prose, then a tag with no blank line between them, which stays prose:
<Term id="not-a-tag" means="prose" />

An inline <Rule> inside a paragraph is prose too.

## Values

<Figure
id="flow"
form="flow"
shows={ref("forms#values")}
caption="One thing becomes another."
parts={[{ name: "draft", to: "checks" }, { "name": "page", n: 2, at: "xy" },]}
weight="lead"
/>

<Lens id="map" of="conviction" />

<Group id="empty" title="An empty group" intent="prove" />

#### A fourth-level heading is prose

```markdown
<Term id="in-a-fence" means="a fence keeps a tag as text" />
```

~~~~
<Group id="fenced" title="Fenced" intent="tell">
~~~~

<Term id="spaced" means="a value set apart"   not  =  "by spaces"   retired={null} code={-1.5} />
