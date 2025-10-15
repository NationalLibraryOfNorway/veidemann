import {ChangeDetectionStrategy, Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule} from '@angular/material/dialog';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-crawlerstatus-dialog',
  templateUrl: './crawlerstatus-dialog.component.html',
  styleUrls: ['./crawlerstatus-dialog.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatDialogModule
  ],
  standalone: true
})
export class CrawlerStatusDialogComponent {
  shouldPause: boolean;

  constructor(@Inject(MAT_DIALOG_DATA) public data: any) {
    this.shouldPause = data.shouldPause;
  }

}
