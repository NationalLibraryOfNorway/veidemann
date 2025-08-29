import {Component, Inject, OnInit} from '@angular/core';
import {CrawlConfigDetailsComponent} from '..';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {MAT_DIALOG_DATA, MatDialogContent, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject} from '../../../../../shared/models/config';
import {MetaComponent} from '../../meta/meta.component';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {MatCheckbox} from '@angular/material/checkbox';

@Component({
  selector: 'app-crawlconfig-dialog',
  templateUrl: './crawlconfig-dialog.component.html',
  styleUrls: ['./crawlconfig-dialog.component.css'],
  imports: [
    MatCheckbox,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    MetaComponent,
    ReactiveFormsModule,

  ],
  standalone: true
})
export class CrawlConfigDialogComponent extends CrawlConfigDetailsComponent implements OnInit {

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) public data: ConfigDialogData,
              public dialogRef: MatDialogRef<CrawlConfigDialogComponent>) {
    super(fb, authService);
    this.createForm();
    this.configObject = this.data.configObject;
    this.collections = this.data.options.collections;
    this.browserConfigs = this.data.options.browserConfigs;
    this.politenessConfigs = this.data.options.politenessConfigs;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject {
    return this.prepareSave();
  }
}
