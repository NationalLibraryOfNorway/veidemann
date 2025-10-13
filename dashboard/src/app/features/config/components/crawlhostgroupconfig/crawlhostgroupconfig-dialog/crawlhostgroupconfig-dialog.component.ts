import {Component, Inject, OnInit} from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject} from '../../../../../shared/models/config';
import {CrawlHostGroupConfigDetailsComponent} from '..';
import {MetaComponent} from '../../meta/meta.component';
import {DurationPickerComponent} from '../../durationpicker/duration-picker';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIcon} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatCardModule} from '@angular/material/card';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';
import {MatButtonModule} from '@angular/material/button';

@Component({
  selector: 'app-crawlhostgroupconfig-dialog',
  templateUrl: './crawlhostgroupconfig-dialog.component.html',
  styleUrls: ['./crawlhostgroupconfig-dialog.component.css'],
  imports: [
    DurationPickerComponent,
    FlexDirective,
    LayoutDirective,
    LayoutGapDirective,
    MatButtonModule,
    MatCardModule,
    MatDialogModule,
    MatFormFieldModule,
    MatIcon,
    MatInputModule,
    MetaComponent,
    ReactiveFormsModule
  ],
  standalone: true
})
export class CrawlHostGroupConfigDialogComponent extends CrawlHostGroupConfigDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<CrawlHostGroupConfigDialogComponent>) {
    super(fb, authService);
    this.createForm();
    this.configObject = this.data.configObject;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }
}
