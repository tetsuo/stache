const t = require('./types')
const { Section, InvertedSection, Variable, Comment, Text, Element } = require('./domain')

const decode = u => {
  if (t.UnknownList(u)) {
    if (u.length < 1) {
      throw new TypeError('expected a non-empty array')
    }
    const [_0, _1, _2] = u
    if (t.String(_0)) {
      // element?
      if (_0.trim().length < 1) {
        throw new TypeError('expected an element name, got empty string')
      }
      const props = {}
      if (u.length > 1) {
        // has props?
        if (!t.UnknownStruct(_1)) {
          throw new TypeError(`${_0}: expected a props object, got ${JSON.stringify(_1)}`)
        }
        const keys = Object.keys(_1)
        let key
        let i = 0
        for (; i < keys.length; i++) {
          key = keys[i]
          if (!t.UnknownList(_1[key])) {
            throw new TypeError(`${_0} > ${key}: expected an array of child nodes, got ${JSON.stringify(_1[key])}`)
          }
          let j = 0
          const propChildren = []
          const arr = _1[key]
          for (; j < arr.length; j++) {
            propChildren.push(decode(arr[j]))
          }
          props[key] = propChildren
        }
      }
      const children = []
      if (u.length > 2) {
        // has children?
        if (!t.UnknownList(_2)) {
          throw new TypeError(`${_0}: expected an array of child nodes, got ${JSON.stringify(_2)}`)
        }
        for (let i = 0; i < _2.length; i++) {
          children.push(decode(_2[i]))
        }
      }
      return Element(_0, props, children)
    } else if (t.Integer(_0) && _0 >= 2 && _0 <= 5) {
      if (_0 === 2 || _0 === 4) {
        if (!t.NonEmptyString(_1)) {
          throw new TypeError(`expected a non-empty string as section name, got ${JSON.stringify(_1)}`)
        }
        const children = []
        if (u.length > 2) {
          for (let i = 0; i < _2.length; i++) {
            children.push(decode(_2[i]))
          }
        }
        if (_0 === 2) {
          return Section(_1, children)
        }
        return InvertedSection(_1, children)
      } else if (_0 === 3) {
        if (!t.NonEmptyString(_1)) {
          throw new TypeError(`expected a non-empty string as variable name, got ${JSON.stringify(_1)}`)
        }
        return Variable(_1)
      } else {
        // implicitly, if (_0 === 5)
        if (!t.String(_1)) {
          throw new TypeError(`expected a string as comment, got ${JSON.stringify(_1)}`)
        }
        return Comment(_1)
      }
    }
    throw new TypeError(`expected a tag name or a valid id, got ${JSON.stringify(_0)}`)
  }
  if (t.String(u)) {
    return Text(u)
  }
  throw new TypeError(`expected an array or string, got ${JSON.stringify(u)}`)
}

module.exports = decode
