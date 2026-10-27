// Resaltado de sintaxis mínimo y SEGURO: todo el texto se escapa antes de insertarse como HTML.
const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

const TSP_TYPES = 'string|boolean|bytes|int8|int16|int32|int64|safeint|integer|uint8|uint16|uint32|uint64|float|float32|float64|numeric|decimal|decimal128|url|utcDateTime|offsetDateTime|plainDate|plainTime|duration|Record|Array'
const RULES = {
  tsp: [
    [/\/\/[^\n]*/y, 'c'], [/\/\*[\s\S]*?(?:\*\/|$)/y, 'c'], [/"(?:\\.|[^"\\\n])*"/y, 's'], [/@[A-Za-z_][\w.]*/y, 'd'],
    [/\b(?:model|scalar|enum|union|interface|namespace|import|using|extends|is|op|alias|extern|dec|const|valueof|void|never|unknown|true|false|null)\b/y, 'k'],
    [new RegExp(`\\b(?:${TSP_TYPES})\\b`, 'y'), 't'], [/\b\d+(?:\.\d+)?\b/y, 'n'],
  ],
  json: [
    [/"(?:\\.|[^"\\\n])*"(?=\s*:)/y, 'p'], [/"(?:\\.|[^"\\\n])*"/y, 's'], [/\b(?:true|false|null)\b/y, 'k'], [/-?\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b/y, 'n'],
  ],
  xml: [
    [/<!--[\s\S]*?(?:-->|$)/y, 'c'], [/<\?[\s\S]*?(?:\?>|$)/y, 'c'], [/<\/?[A-Za-z_][\w:.-]*/y, 'k'], [/\/?>/y, 'k'], [/"[^"]*"/y, 's'], [/\b[A-Za-z_:][\w:.-]*(?==)/y, 'd'],
  ],
  proto: [
    [/\/\/[^\n]*/y, 'c'], [/\/\*[\s\S]*?(?:\*\/|$)/y, 'c'], [/"(?:\\.|[^"\\\n])*"/y, 's'],
    [/\b(?:syntax|package|import|message|enum|service|rpc|returns|repeated|optional|oneof|map|option|reserved)\b/y, 'k'],
    [/\b(?:string|bool|bytes|int32|int64|uint32|uint64|sint32|sint64|fixed32|fixed64|float|double)\b/y, 't'], [/\b\d+\b/y, 'n'],
  ],
}

/** Devuelve un arreglo de líneas, cada una como HTML ya escapado. */
export function highlightLines(text, lang) {
  const rules = RULES[lang]
  const lines = ['']
  const push = (piece, cls) => {
    const parts = piece.split('\n')
    parts.forEach((p, i) => {
      if (i > 0) lines.push('')
      if (p) lines[lines.length - 1] += cls ? `<span class="tk-${cls}">${esc(p)}</span>` : esc(p)
    })
  }
  if (!rules || text.length > 300000) { push(text, null); return lines }
  let i = 0, plain = ''
  while (i < text.length) {
    let hit = null
    for (const [re, cls] of rules) {
      re.lastIndex = i
      const m = re.exec(text)
      if (m && m[0].length) { hit = [m[0], cls]; break }
    }
    if (hit) {
      if (plain) { push(plain, null); plain = '' }
      push(hit[0], hit[1])
      i += hit[0].length
    } else { plain += text[i]; i++ }
  }
  if (plain) push(plain, null)
  return lines
}

export const LANG_OF = { tsp: 'tsp', 'tsp-file': 'tsp', jsonschema: 'json', openapi: 'json', xsd: 'xml', proto: 'proto' }
