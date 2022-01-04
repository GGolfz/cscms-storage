import { Fragment } from 'react'
import FileIcon from '../util/FileIcon'
import { formatFileSize } from '../../utils/formatFileSize'
import { formatDate } from '../../utils/formatText'
import Icon from '../util/Icon'
import styles from '../../styles/file/Detail.module.css'
import Swal from 'sweetalert2'
import axios from 'axios'

const FileDetail = ({fetchFiles, file, setError }) => {
	const copyToClipboard = () => {
		var copyText = document.createElement('input')
		copyText.setAttribute('value', file.url)
		document.body.appendChild(copyText)
		copyText.select()
		copyText.setSelectionRange(0, 99999)
		navigator.clipboard.writeText(copyText.value)
		document.body.removeChild(copyText)
	}
	const handleEdit = value => {
		axios
			.patch(`https://storage.cscms.me/api/file/${file.id}?token=${value}`)
			.then(() => {
				fetchFiles();
				Swal.fire({
					title: 'Updated!',
					text: 'Your file was successfully updated',
					type: 'success'
				})
			})
			.catch(err => {
				setError(err.response.data.message)
			})
	}
	const handleClickEdit = () => {
		Swal.fire({
			title: 'Edit slug',
			input: 'text',
			inputValue: file.token,
			inputValidator: value => {
				if (!value) {
					return 'Slug is required'
				}
			},
			confirmButtonText: 'Update',
			showCancelButton: true,
			reverseButtons: true,
			cancelButtonText: 'Cancel'
		}).then(result => {
			if (result.isConfirmed) {
				handleEdit(result.value)
			}
		})
	}
	const handleDelete = () => {
		axios
			.delete(`https://storage.cscms.me/api/file/${file.id}`)
			.then(() => {
				fetchFiles();
				Swal.fire({
					title: 'Deleted!',
					text: 'Your file was successfully deleted',
					type: 'success'
				})
			})
			.catch(err => {
				setError(err.response.data.message)
			})
	}
	const handleClickDelete = () => {
		Swal.fire({
			title: 'Do you want to delete file?',
			text: "You won't be able to revert this!",
			confirmButtonText: 'Delete',
			showCancelButton: true,
			reverseButtons: true,
			cancelButtonText: 'Cancel',
			confirmButtonColor: '#dc3545'
		}).then(result => {
			if (result.isConfirmed) {
				handleDelete()
			}
		})
	}
	return (
		<Fragment>
			<td>
				<FileIcon ext={file.filename.split('.')[1]} type={file.file_type.split('/')[0]} />{' '}
				{file.filename}
			</td>
			<td>{formatFileSize(file.file_size)}</td>
			<td>{formatDate(file.updated_at)}</td>
			<td>
				<div className={styles.ActionList}>
					<div onClick={handleClickEdit}>
						<Icon name="edit" />
					</div>
					<div onClick={handleClickDelete}>
						<Icon name="delete" />
					</div>
					<div onClick={copyToClipboard}>
						<Icon name="copy" />
					</div>
				</div>
			</td>
		</Fragment>
	)
}

export default FileDetail
