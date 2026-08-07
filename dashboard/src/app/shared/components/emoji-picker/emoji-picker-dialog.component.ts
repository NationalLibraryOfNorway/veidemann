import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatDialogModule, MatDialogRef} from '@angular/material/dialog';

import {EmojiPickerComponent} from './emoji-picker.component';

@Component({
  selector: 'app-emoji-picker-dialog',
  templateUrl: './emoji-picker-dialog.component.html',
  styleUrls: ['./emoji-picker-dialog.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [EmojiPickerComponent, MatButtonModule, MatDialogModule],
  standalone: true,
})
export class EmojiPickerDialogComponent {
  private readonly dialogRef = inject<MatDialogRef<EmojiPickerDialogComponent, string>>(MatDialogRef);

  protected selectEmoji(unicode: string): void {
    this.dialogRef.close(unicode);
  }
}
