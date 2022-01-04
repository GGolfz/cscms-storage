import { Box, LinearProgress } from '@material-ui/core'
import React, { Fragment } from 'react'
import { useDropzone } from 'react-dropzone'
import styles from '../../styles/upload/Dropzone.module.css'
const DropZone = ({ onDrop, selectedFilename, type, progress }) => {
	const { getRootProps, getInputProps, isDragActive } = useDropzone({
		onDrop,
		maxFiles: 1,
		multiple: false,
		maxSize: type === 'file' ? 100 << 20 : (100 << 20) / 20
	})

	const renderContainerText = () => {
		if (isDragActive) {
			return (
				<Fragment>
					<img src="dropfile.png" alt="dropfile" />
					<h2 className={styles.dropfileText}>Drop right here !!!</h2>
				</Fragment>
			)
		}
		if (type === 'file') {
			return (
				<Fragment>
					<img src="file.png" alt="file-icon" />
					<h2>Drag and Drop your file</h2>
					{progress !== -1 ? (
						renderProgressBar()
					) : (
						<Fragment>
							<p>Drop your file here, or click to select file</p>
							<p>The maximum file size is 100MB</p>
						</Fragment>
					)}
				</Fragment>
			)
		} else if (type === 'image') {
			return (
				<Fragment>
					<img src="image.png" alt="img-icon" />
					<h2>Drag and Drop your image</h2>
					{progress !== -1 ? (
						renderProgressBar()
					) : (
						<Fragment>
							<p>Drop your image here, or click to select image</p>
							<p>The maximum file size is 5MB</p>
						</Fragment>
					)}
				</Fragment>
			)
		}
	}
	const getClassName = () => {
		if (isDragActive) {
			return styles.drag
		}
		if (selectedFilename.length !== 0) {
			return styles.active
		}
		return ''
	}

	const renderProgressBar = () => {
		return (
			<Fragment>
				<Box sx={{ display: 'flex', alignItems: 'center', width: '600px' }}>
					<Box sx={{ width: '100%', mr: 1 }}>
						<LinearProgress variant="determinate" value={progress} />
					</Box>
					<Box sx={{ minWidth: 100 }}>
						{`${progress === 100 ? 'Encrypting...' : 'Uploading...'}`}
					</Box>
				</Box>
			</Fragment>
		)
	}

	return (
		<div className={`${styles.Dropzone} ${getClassName()}`} {...getRootProps()}>
			<input className="dropzone-input" {...getInputProps()} />
			<div className={styles.DropZoneTextContainer}>{renderContainerText()}</div>
		</div>
	)
}

export default DropZone
