import { Fragment, useEffect, useState } from 'react'
import styles from '../../styles/file/FileList.module.css'
import Icon from '../util/Icon'
import axios from 'axios'
import FileDetail from './FileDetail'
import ImageDetail from './ImageDetail'
const FileList = ({ setError }) => {
	const [sort, setSort] = useState({ name: '', asc: true })
	const [files, setFiles] = useState([])
	const [displayFile, setDisplayFile] = useState([])
	useEffect(() => {
		fetchFiles()
	}, [])
	const fetchFiles = async () => {
		const fileRes = await axios.get('https://storage.cscms.me/api/file')
		const fileData = fileRes.data.map(file => ({
			...file,
			type: 'file',
			url: 'https://storage.cscms.me/' + file.token
		}))
		const imageRes = await axios.get('https://storage.cscms.me/api/image')
		const imageData = imageRes.data.map(image => ({
			...image,
			type: 'image',
			url: 'https://img.cscms.me/' + image.file_path,
			file_type: 'image',
			filename: image.original_filename
		}))
		setFiles([...fileData, ...imageData])
	}
	useEffect(() => {
		setDisplayFile(files)
		setSort({ name: '', asc: true })
	}, [files])
	useEffect(() => {
		if (sort.name === '') {
			setDisplayFile(files)
			setSort({ name: '', asc: true })
		} else {
			let temp = [...files].sort((a, b) => {
				if (sort.name === 'size') {
					if (sort.asc) return a.size - b.size
					return b.size - a.size
				}
				if (sort.asc) {
					return a[sort.name] > b[sort.name] ? 1 : -1
				} else {
					return b[sort.name] > a[sort.name] ? 1 : -1
				}
			})
			setDisplayFile(temp)
		}
	}, [sort])
	const handleSort = type => {
		if (type === sort.name) {
			if (sort.asc) {
				setSort({ name: type, asc: false })
			} else {
				setSort({ name: '', asc: true })
			}
		} else {
			setSort({ name: type, asc: true })
		}
	}

	return (
		<Fragment>
			<div className={styles.FileListWrapper}>
				<h3>My Files</h3>
				<div>
					<table className={styles.FileList}>
						<thead>
							<tr>
								<th style={{ width: '45%' }}>
									<div onClick={() => handleSort('filename')}>
										Name{' '}
										{sort.name === 'filename' && sort.asc ? (
											<Icon name="arrow-down" />
										) : (
											<Icon name="arrow-up" />
										)}
									</div>
								</th>
								<th style={{ width: '15%' }}>
									<div onClick={() => handleSort('file_size')}>
										Size{' '}
										{sort.name === 'file_size' && sort.asc ? (
											<Icon name="arrow-down" />
										) : (
											<Icon name="arrow-up" />
										)}
									</div>
								</th>
								<th style={{ width: '25%' }}>
									<div onClick={() => handleSort('updated_at')}>
										Last Modified{' '}
										{sort.name === 'updated_at' && sort.asc ? (
											<Icon name="arrow-down" />
										) : (
											<Icon name="arrow-up" />
										)}
									</div>
								</th>
								<th style={{ width: '15%' }}></th>
							</tr>
						</thead>
						<tbody>
							{displayFile.length === 0 ? (
								<Fragment>
									<tr>
										<td className={styles.Empty} colSpan={4}>
											No files found
										</td>
									</tr>
								</Fragment>
							) : (
								displayFile.map((file, index) => {
									return (
										<tr key={index} className={styles.Row}>
											{file.type === 'file' ? (
												<FileDetail
													fetchFiles={fetchFiles}
													setError={setError}
													file={file}
												/>
											) : (
												<ImageDetail
													fetchFiles={fetchFiles}
													setError={setError}
													file={file}
												/>
											)}
										</tr>
									)
								})
							)}
						</tbody>
					</table>
				</div>
			</div>
		</Fragment>
	)
}

export default FileList
