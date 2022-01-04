import { Fragment } from 'react'
import Icon from './Icon'
import styles from '../../styles/util/FileIcon.module.css'

const FileIcon = ({ ext, type }) => {
	const getDisplay = () => {
		if (type === 'image') {
			return <Icon name="img-file" />
		} else if (type === 'video') {
			return <Icon name="video-file" />
		} else if (type === 'audio') {
			return <Icon name="music-file" />
		} else if (ext === 'PDF') {
			return <Icon name="pdf-file" />
		} else if (ext === 'DOCX' || ext === 'DOC') {
			return <Icon name="document-file" />
		} else if (ext === 'XLSX' || ext === 'XLS') {
			return <Icon name="sheet-file" />
		} else if (ext === 'PPTX' || ext === 'PPT') {
			return <Icon name="slide-file" />
		} else if (ext === 'PSD') {
			return <Icon name="ps-file" />
		} else if (ext === 'AI') {
			return <Icon name="ai-file" />
		} else {
			return <Icon name="other-file" />
		}
	}
	return (
		<Fragment>
			<div className={styles.FileIcon}
			>
				{getDisplay()}
			</div>
		</Fragment>
	)
}

export default FileIcon
