import { useState, useEffect, Fragment } from 'react'
import ReactGA from 'react-ga'
import Navbar from './components/layout/Navbar'
import Sidebar from './components/layout/Sidebar'
import AuthForm from './components/auth/AuthForm'
import styles from './styles/App.module.css'
import { Dialog } from '@material-ui/core'
import UploadContainer from './components/upload/UploadContainer'
import { useRecoilState } from 'recoil'
import { authState } from './store/auth'
import axios from 'axios'
import Swal from 'sweetalert2'
import FileList from './components/file/FileList'
import UserProfile from './components/auth/UserProfile'
import { formatFileSize } from './utils/formatFileSize'
import { formatDate } from './utils/formatText'
function App() {
	const [route, setRoute] = useState('file')
	const [auth, setAuth] = useRecoilState(authState)
	const [dialog, setDialog] = useState(null)
	const [progress, setProgress] = useState(-1)
	const [error, setError] = useState(null)
	useEffect(() => {
		ReactGA.initialize('G-S7NPY62JTS')
		ReactGA.pageview(window.location.pathname)
	}, [])
	useEffect(() => {
		axios
			.get('https://storage.cscms.me/auth/user')
			.then(res => {
				setAuth({
					isAuthenticated: true,
					user: {
						name: res.data.username,
						image: res.data.avatar_url,
						email: res.data.email
					}
				})
			})
			.catch(() => {
				setAuth({
					isAuthenticated: false,
					user: null
				})
			})
	}, [setAuth])
	useEffect(() => {
		if (auth.isAuthenticated) {
			setDialog(null)
		}
	}, [auth])

	useEffect(() => {
		if (error != null) {
			Swal.fire({
				title: 'An error occured',
				text: error,
				icon: 'error',
				confirmButtonText: 'Ok'
			})
			setError(null)
		}
	}, [error])

	const handleAction = action => {
		switch (action) {
			case 'signup':
				setDialog('signup')
				break
			case 'login':
				setDialog('login')
				break
			case 'logout':
				axios
					.get('https://storage.cscms.me/auth/logout')
					.then(() => {
						setAuth(false)
					})
					.catch(err => {
						setError(err.response.data.message)
					})
				break
			default:
				break
		}
	}
	const handleChangeRoute = newRoute => {
		if (newRoute === route) return
		setRoute(newRoute)
	}
	const handleUpload = data => {
		try {
			if (route === 'file') {
				handleUploadFile(data)
			} else if (route === 'image') {
				handleUploadImage(data)
			}
		} catch (err) {
			throw err
		}
	}
	const handleUploadFile = async ({ selectedFile, slug, duration, clearFile }) => {
		const formdata = new FormData()
		formdata.append('file', selectedFile)

		try {
			const res = await axios.post('https://storage.cscms.me/api/file', formdata, {
				onUploadProgress: progressEvent => {
					const uploadPercent = Math.round(
						(progressEvent.loaded / progressEvent.total) * 100
					)
					setProgress(uploadPercent)
				},
				params: { slug, duration }
			})
			setProgress(-1)
			clearFile()
			const { token, filename, file_size, expired_at } = res.data
			var html =
				'<div style="text-align:left">Download URL: https://storage.cscms.me/' +
				token +
				'<br>File name: ' +
				filename +
				'<br>File size: ' +
				formatFileSize(file_size) +
				'<br>Valid Though: ' +
				formatDate(expired_at) +
				'</div>'
			Swal.fire({
				title: 'Upload File Success',
				icon: 'success',
				html,
				showCancelButton: true,
				confirmButtonText: 'Copy URL',
				cancelButtonText: 'Close'
			}).then(result => {
				if (result.isConfirmed) {
					var copyText = document.createElement('input')
					copyText.setAttribute('value', 'https://storage.cscms.me/' + token)
					document.body.appendChild(copyText)
					copyText.select()
					copyText.setSelectionRange(0, 99999)
					navigator.clipboard.writeText(copyText.value)
					document.body.removeChild(copyText)
				}
			})
			ReactGA.event({
				category: 'file',
				action: 'Upload file',
				value: selectedFile.size
			})
		} catch (err) {
			throw err
		}
	}
	const handleUploadImage = async ({ selectedFile, clearFile }) => {
		const formdata = new FormData()
		formdata.append('image', selectedFile)

		try {
			const res = await axios.post('https://storage.cscms.me/api/image', formdata, {
				onUploadProgress: progressEvent => {
					const uploadPercent = Math.round(
						(progressEvent.loaded / progressEvent.total) * 100
					)
					setProgress(uploadPercent)
				}
			})
			setProgress(-1)
			clearFile()
			const { file_path, original_filename, file_size } = res.data
			var html =
				'<div style="text-align:left">URL: https://img.cscms.me/' +
				file_path +
				'<br>File name: ' +
				original_filename +
				'<br>File size: ' +
				formatFileSize(file_size) +
				'</div>'
			Swal.fire({
				title: 'Upload Image Success',
				icon: 'success',
				html,
				width: '600px',
				showCancelButton: true,
				confirmButtonText: 'Copy URL',
				cancelButtonText: 'Close'
			}).then(result => {
				if (result.isConfirmed) {
					var copyText = document.createElement('input')
					copyText.setAttribute('value', 'https://img.cscms.me/' + file_path)
					document.body.appendChild(copyText)
					copyText.select()
					copyText.setSelectionRange(0, 99999)
					navigator.clipboard.writeText(copyText.value)
					document.body.removeChild(copyText)
				}
			})
			ReactGA.event({
				category: 'image',
				action: 'Upload Image',
				value: selectedFile.size
			})
		} catch (err) {
			throw err
		}
	}
	const renderScreen = () => {
		switch (route) {
			case 'file':
			case 'image':
				return (
					<UploadContainer
						type={route}
						handleUpload={handleUpload}
						setError={setError}
						progress={progress}
					/>
				)
			case 'myfile':
				if (auth.isAuthenticated) return <FileList setError={setError} />
				setRoute('file')
				break
			default:
				setRoute('file')
				break
		}
	}
	return (
		<Fragment>
			<div className={styles.App}>
				<div>
					<Navbar auth={auth.isAuthenticated} handleAction={handleAction} />
					<div className={styles.ContentWrapper}>
						{auth.isAuthenticated ? (
							<UserProfile user={auth.user} handleChangeRoute={handleChangeRoute} />
						) : null}

						{renderScreen()}
					</div>
					<Sidebar currentRoute={route} handleChangeRoute={handleChangeRoute} />
					{!auth.isAuthenticated && dialog ? (
						<Dialog open={dialog !== null} onClose={() => setDialog(null)}>
							<AuthForm mode={dialog} changeMode={mode => setDialog(mode)} />
						</Dialog>
					) : null}
				</div>
			</div>
		</Fragment>
	)
}

export default App
