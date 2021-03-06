import {Component} from 'react'

export type Props = {
  username: string
  allowDeleteForever: boolean
  setAllowDeleteAccount: (allow: boolean) => void
  onCancel: () => void
  onDeleteForever: () => void
}

export default class DeleteConfirm extends Component<Props> {}
