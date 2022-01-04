const dayjs = require("dayjs")
const localizedFormat = require("dayjs/plugin/localizedFormat")

exports.toTitleCase = string => {
	return string.charAt(0).toUpperCase() + string.slice(1)
}
exports.formatDate = date => {
	dayjs.extend(localizedFormat)
	return dayjs(date).format('LLL')
}